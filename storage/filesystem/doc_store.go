package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

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

func (h *DocStore) loadJSON(path string) (*sent.Doc, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("IO error: %w", err)
	}

	var doc sent.Doc
	err = json.Unmarshal(f, &doc)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding error: %w", err)
	}

	return &doc, nil
}

// LoadNLP preloads docs into memory and injects metadata.
// If docID is provided, only that specific doc is loaded.
// If labels is not empty, only docs matching ALL labels are loaded.
func (h *DocStore) LoadNLP(labels []string, docID *int, cb func(current, total int, name string)) error {
	total := len(h.docs)
outer:
	for i := range h.docs {
		doc := &h.docs[i]

		if docID != nil  {
		    if *docID == i {

                fullPath := filepath.Join(h.docsPath, doc.Title)
                fullDoc, err := h.loadJSON(fullPath)
                if err != nil {
                    return err
                }

                // Metadata Injection
                fullDoc.Id = i
                fullDoc.Title = doc.Title
                for j := range fullDoc.Sentences {
                    fullDoc.Sentences[j].DocId = i
                }

                return nil
            } 

            continue
		}

        // labels and docid filter are exclusive
		for _, req := range labels {
			if !slices.Contains(doc.Labels, req) {
				continue outer
			}
		}

		if cb != nil {
			cb(i+1, total, doc.Title)
		}

		fullPath := filepath.Join(h.docsPath, doc.Title)
		fullDoc, err := h.loadJSON(fullPath)
		if err != nil {
			return err
		}

		// Metadata Injection
		fullDoc.Id = i
		fullDoc.Title = doc.Title
		for j := range fullDoc.Sentences {
			fullDoc.Sentences[j].DocId = i
		}

		doc.Sentences = fullDoc.Sentences
	}

	return nil
}

// List returns the metadata of documents.
// Discovery mode: returns docs where at least one label contains the labelMatch substring.
func (h *DocStore) List(labelSubStr string) ([]sent.Doc, error) {
	if labelSubStr == "" {
		return h.docs, nil
	}

	var filtered []sent.Doc
	for _, doc := range h.docs {
		if labelSubStr != "" {
			if !SliceElementsContains(doc.Labels, labelSubStr) {
				continue
			}
		}
		filtered = append(filtered, doc)
	}
	return filtered, nil
}

func SliceElementsContains(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func (h *DocStore) Read(id int) (sent.Doc, error) {
	if id < 0 || id >= len(h.docs) {
		return sent.Doc{}, fmt.Errorf("doc id out of range: %d", id)
	}

	return h.docs[id], nil
}

// FindCandidates returns ALL sentences from memory if they match the labels.
func (h *DocStore) FindCandidates(lemmas []string, labels []string, after storage.Cursor, limit int, onCandidate func(sent.Sentence) error) (storage.Cursor, error) {

	// If cursor > 0, we already returned everything (EOF).
	if after > 0 {
		return after, nil
	}

	for _, doc := range h.docs {
		for _, s := range doc.Sentences {
			if err := onCandidate(s); err != nil {
				return after, err
			}
		}
	}

	return 1, nil
}

// Labels returns all unique labels found across all documents, sorted alphabetically.
// Discovery mode: if labelSubStr is not empty, returns labels that contain the labelSubStr substring.
func (h *DocStore) Labels(labelSubStr string) ([]string, error) {
	labelMap := make(map[string]bool)
	for _, doc := range h.docs {
		for _, label := range doc.Labels {
			if labelSubStr != "" {
				if !strings.Contains(label, labelSubStr) {
					continue
				}
			}
			labelMap[label] = true
		}
	}

	labels := make([]string, 0, len(labelMap))
	for label := range labelMap {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels, nil
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
