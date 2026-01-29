package main

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func docCommand(opts DocOptions, arg string, isArgFile bool, isRepoFile bool, ui UI) error {
	if isArgFile {
		return renderFile(arg, opts, ui)
	}

	var repo storage.DocRepository
	if isRepoFile {
		pool, err := zombiezen.NewPool(opts.DocPath)
		if err != nil {
			return err
		}
		defer pool.Close()
		repo = zombiezen.NewDocStore(pool)
	} else {
		h, err := filesystem.NewDocStore(opts.DocPath)
		if err != nil {
			return err
		}
		if arg == "" {
			if err := h.LoadList(); err != nil {
				return err
			}
		} else {
			if err := h.LoadList(); err != nil {
				return err
			}
			if err := h.LoadAll(nil); err != nil {
				return err
			}
		}
		repo = h
	}

	if arg == "" {
		return listDocs(repo, ui)
	}

	id, _ := strconv.Atoi(arg)
	doc, err := repo.Read(id)
	if err != nil {
		return err
	}

	renderDoc(doc, opts, ui)
	return nil
}

func renderFile(path string, opts DocOptions, ui UI) error {
	doc, err := filesystem.ReadDoc(path)
	if err != nil {
		absPath, _ := filepath.Abs(path)
		return fmt.Errorf("filesystem document %q: %w", absPath, err)
	}

	renderDoc(doc, opts, ui)
	return nil
}

func renderDoc(doc sent.Doc, opts DocOptions, ui UI) {
	start := opts.Start
	if start < 0 {
		start = 0
	}
	if start >= len(doc.Tokens) {
		return
	}

	sentences := doc.Tokens[start:]
	if opts.Count >= 0 && opts.Count < len(sentences) {
		sentences = sentences[:opts.Count]
	}

	r := render.NewRenderer()
	r.HasColor = false
	for i, sentence := range sentences {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence, prefix)
	}
}

func listDocs(repo storage.DocReader, ui UI) error {
	docs, err := repo.List()
	if err != nil {
		return err
	}

	for _, doc := range docs {
		fmt.Fprintf(ui.Out, "📖 %d %s\n", doc.Id, doc.Title)
	}
	return nil
}
