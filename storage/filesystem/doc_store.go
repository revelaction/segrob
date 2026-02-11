package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// Preload preloads docs into memory.
// If labels is not empty, only docs matching ALL labels are loaded.
func (h *DocStore) Preload(labels []string, cb func(current, total int, name string)) error {
	total := len(h.docs)
	for i := range h.docs {
		doc := &h.docs[i]

		if len(labels) > 0 && !matchLabels(doc.Labels, labels) {
			continue
		}

		if cb != nil {
			cb(i+1, total, doc.Title)
		}

		fullDoc, err := h.Read(doc.Id)
		if err != nil {
			return err
		}

		doc.Sentences = fullDoc.Sentences
	}

	return nil
}

func (h *DocStore) List(labelMatch string) ([]sent.Doc, error) {
	if labelMatch == "" {
		return h.docs, nil
	}

	var filtered []sent.Doc
	for _, doc := range h.docs {
		if labelMatch != "" {
			if !SliceElementsContains(doc.Labels, labelMatch) {
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

func (h *DocStore) Labels(pattern string) ([]string, error) {
	labelMap := make(map[string]bool)
	for _, doc := range h.docs {
		for _, label := range doc.Labels {
			if pattern != "" {
				if !strings.Contains(label, pattern) {
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
