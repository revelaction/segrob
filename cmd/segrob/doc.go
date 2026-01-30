package main

import (
	"fmt"
	"os"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func docCommand(opts DocOptions, id int, ui UI) error {
	info, err := os.Stat(opts.DocPath)
	if err != nil {
		return fmt.Errorf("repository not found: %s", opts.DocPath)
	}

	var repo storage.DocRepository
	if info.IsDir() {
		h, err := filesystem.NewDocStore(opts.DocPath)
		if err != nil {
			return err
		}
		repo = h
	} else {
		pool, err := zombiezen.NewPool(opts.DocPath)
		if err != nil {
			return err
		}
		defer pool.Close()
		repo = zombiezen.NewDocStore(pool)
	}

	doc, err := repo.Read(id)
	if err != nil {
		return err
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
	if opts.Count != nil {
		limit := *opts.Count
		if limit < len(sentences) {
			sentences = sentences[:limit]
		}
	}

	r := render.NewRenderer()
	r.HasColor = false
	for i, sentence := range sentences {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence, prefix)
	}
}
