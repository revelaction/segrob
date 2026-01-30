package main

import (
	"fmt"
	"os"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func sentenceCommand(opts SentenceOptions, docId int, sentId int, ui UI) error {
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

	doc, err := repo.Read(docId)
	if err != nil {
		return err
	}

	if sentId < 0 || sentId >= len(doc.Tokens) {
		return fmt.Errorf("sentence index %d out of bounds (0-%d)", sentId, len(doc.Tokens)-1)
	}

	s := doc.Tokens[sentId]
	r := render.NewRenderer()
	r.HasColor = false
	prefix := fmt.Sprintf("✍  %d ", sentId)
	r.Sentence(s, prefix)
	fmt.Fprintln(ui.Out)

	for _, token := range s {
		fmt.Fprintf(ui.Out, "%20q %15q %8s %6d %6d %8s %s\n", token.Text, token.Lemma, token.Pos, token.Id, token.Head, token.Dep, token.Tag)
	}

	return nil
}
