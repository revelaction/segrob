package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

type DocStore struct {
	docsPath string

	// In-memory cache
	docs []sent.Doc
}

var _ storage.DocRepository = (*DocStore)(nil)

// NewDocStore creates a filesystem document handler.
func NewDocStore(path string) (*DocStore, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("filesystem doc store requires a directory, got file: %s", path)
	}

	docsPath := path
	files, err := os.ReadDir(docsPath)
	if err != nil {
		return nil, err
	}

	docs := make([]sent.Doc, 0, len(files))
	idx := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			labels, err := readLabels(filepath.Join(docsPath, file.Name()))
			if err != nil {
				// handle or skip
			}
			docs = append(docs, sent.Doc{
				Id:     idx,
				Title:  file.Name(),
				Labels: labels,
			})
			idx++
		}
	}

	return &DocStore{docsPath: docsPath, docs: docs}, nil
}

// Preload preloads all docs into memory.
func (h *DocStore) Preload(cb func(current, total int, name string)) error {
	if h.docs == nil {
		return fmt.Errorf("docs not initialized")
	}

	total := len(h.docs)
	for i := range h.docs {
		doc := &h.docs[i] // pointer to modify in place

		if cb != nil {
			cb(i+1, total, doc.Title)
		}

		fullDoc, err := h.Read(i)
		if err != nil {
			return err
		}

		// Copy loaded content into existing metadata struct
		doc.Sentences = fullDoc.Sentences
		doc.Labels = fullDoc.Labels
		// Title and Id are already set
	}

	return nil
}

func (h *DocStore) List() ([]sent.Doc, error) {
	return h.docs, nil
}

func (h *DocStore) Read(id int) (sent.Doc, error) {
	if id < 0 || id >= len(h.docs) {
		return sent.Doc{}, fmt.Errorf("doc id out of range: %d", id)
	}

	meta := h.docs[id]
	fullPath := filepath.Join(h.docsPath, meta.Title)

	f, err := os.ReadFile(fullPath)
	if err != nil {
		return sent.Doc{}, fmt.Errorf("IO error: %w", err)
	}

	var doc sent.Doc
	err = json.Unmarshal(f, &doc)
	if err != nil {
		return sent.Doc{}, fmt.Errorf("JSON decoding error: %w", err)
	}

	// Ensure metadata consistency
	doc.Id = meta.Id
	doc.Title = meta.Title

	return doc, nil
}

// FindCandidates returns ALL sentences from memory if they match the labels.
func (h *DocStore) FindCandidates(lemmas []string, labels []string, after storage.Cursor, limit int, onCandidate func(sent.Sentence) error) (storage.Cursor, error) {
	if len(lemmas) == 0 {
		return after, nil
	}

	// If cursor > 0, we already returned everything (EOF).
	if after > 0 {
		return after, nil
	}

	for _, doc := range h.docs {
		// Skip if document doesn't match all required labels
		if !matchLabels(doc.Labels, labels) {
			continue
		}

		for _, s := range doc.Sentences {
			if err := onCandidate(s); err != nil {
				return after, err
			}
		}
	}

	return 1, nil
}

// Helper to check if docLabels contains all requiredLabels
func matchLabels(docLabels, requiredLabels []string) bool {
	if len(requiredLabels) == 0 {
		return true
	}
	for _, req := range requiredLabels {
		found := false
		for _, doc := range docLabels {
			if doc == req { // Exact match for storage consistency
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (h *DocStore) Write(doc sent.Doc) error {
	return fmt.Errorf("read-only storage")
}


func readLabels(path string) ([]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    dec := json.NewDecoder(f)
    for {
        tok, err := dec.Token()
        if err != nil {
            return nil, err // or break
        }
        key, ok := tok.(string)
        if ok && key == "Labels" {
            var labels []string
            if err := dec.Decode(&labels); err != nil {
                return nil, err
            }
            return labels, nil
        }
    }
}
