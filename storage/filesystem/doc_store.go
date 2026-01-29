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
	docDir string

	// In-memory cache
	docs []sent.Doc
}

var _ storage.DocRepository = (*DocStore)(nil)

// NewDocStore creates a filesystem document handler.
func NewDocStore(docDir string) (*DocStore, error) {
	files, err := os.ReadDir(docDir)
	if err != nil {
		return nil, err
	}

	docs := make([]sent.Doc, 0, len(files))

	idx := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			// TODO: maybe load labels here?
			docs = append(docs, sent.Doc{
				Id:    idx,
				Title: file.Name(),
			})
			idx++
		}
	}

	return &DocStore{
		docDir: docDir,
		docs:   docs,
	}, nil
}

// LoadAll preloads all docs into memory.
func (h *DocStore) LoadAll(cb func(total int, name string)) error {
	if h.docs == nil {
		return fmt.Errorf("docs not initialized")
	}

	total := len(h.docs)
	for i := range h.docs {
		doc := &h.docs[i] // pointer to modify in place

		if cb != nil {
			cb(total, doc.Title)
		}

		fullDoc, err := ReadDoc(filepath.Join(h.docDir, doc.Title))
		if err != nil {
			return err
		}

		// Copy loaded content into existing metadata struct
		doc.Tokens = fullDoc.Tokens
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
	return h.docs[id], nil
}

// FindCandidates returns ALL sentences from memory.
func (h *DocStore) FindCandidates(lemmas []string, after storage.Cursor, limit int) ([]storage.SentenceResult, storage.Cursor, error) {
	// If cursor > 0, we already returned everything (EOF).
	if after > 0 {
		return nil, after, nil
	}

	var results []storage.SentenceResult
	for i, doc := range h.docs {
		for _, tokens := range doc.Tokens {
			results = append(results, storage.SentenceResult{
				RowID:    0,
				DocID:    i,
				DocTitle: doc.Title,
				Tokens:   tokens,
			})
		}
	}

	return results, 1, nil
}

func (h *DocStore) Write(doc sent.Doc) error {
	return fmt.Errorf("read-only storage")
}

// ReadDoc reads a Doc JSON from the given path and unmarshals it.
func ReadDoc(path string) (sent.Doc, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return sent.Doc{}, fmt.Errorf("IO error: %w", err)
	}

	var doc sent.Doc
	err = json.Unmarshal(f, &doc)
	if err != nil {
		return sent.Doc{}, fmt.Errorf("JSON decoding error: %w", err)
	}

	return doc, nil
}
