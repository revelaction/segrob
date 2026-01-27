package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

type DocHandler struct {
	docDir string

	// In-memory cache
	docs     []sent.Doc
	docNames []string
}

var _ storage.DocRepository = (*DocHandler)(nil)

// NewDocHandler creates and LOADS a filesystem document handler.
// The callback is called for each file loaded (total, current_name).
func NewDocHandler(docDir string, cb func(total int, name string)) (*DocHandler, error) {
	h := &DocHandler{
		docDir: docDir,
	}

	files, err := os.ReadDir(h.docDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			names = append(names, file.Name())
		}
	}
	h.docNames = names

	// Preload all docs to match legacy behavior
	h.docs = make([]sent.Doc, 0, len(names))

	total := len(names)
	for i, name := range names {
		if cb != nil {
			cb(total, name)
		}

		content, err := os.ReadFile(filepath.Join(h.docDir, name))
		if err != nil {
			return nil, err
		}

		var doc sent.Doc
		if err := json.Unmarshal(content, &doc); err != nil {
			return nil, err
		}
		doc.Title = name
		doc.Id = i

		h.docs = append(h.docs, doc)
	}

	return h, nil
}

func (h *DocHandler) Names() ([]string, error) {
	return h.docNames, nil
}

func (h *DocHandler) Doc(id int) (sent.Doc, error) {
	if id < 0 || id >= len(h.docs) {
		return sent.Doc{}, fmt.Errorf("doc id out of range: %d", id)
	}
	return h.docs[id], nil
}

func (h *DocHandler) DocForName(name string) (sent.Doc, error) {
	for _, doc := range h.docs {
		if doc.Title == name {
			return doc, nil
		}
	}
	return sent.Doc{}, fmt.Errorf("doc not found: %s", name)
}

// FindCandidates returns ALL sentences from memory.
func (h *DocHandler) FindCandidates(lemmas []string, after storage.Cursor, limit int) ([]storage.SentenceResult, storage.Cursor, error) {
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

func (h *DocHandler) Sentence(rowid int64) ([]sent.Token, error) {
	return nil, fmt.Errorf("Sentence(rowid) not implemented for filesystem")
}

func (h *DocHandler) WriteDoc(doc sent.Doc) error {
	return fmt.Errorf("read-only storage")
}
