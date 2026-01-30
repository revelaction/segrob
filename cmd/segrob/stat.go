package main

import (
	"fmt"
	"os"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func statCommand(opts StatOptions, docId int, sentId *int, ui UI) error {
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

	if sentId != nil {
		if *sentId < 0 || *sentId >= len(doc.Tokens) {
			return fmt.Errorf("sentence index %d out of bounds (doc has %d sentences)", *sentId, len(doc.Tokens))
		}
		doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*sentId]}}
	}

	hdl := stat.NewHandler()
	hdl.Aggregate(doc)

	stats := hdl.Get()
	fmt.Fprintf(ui.Out, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)

	return nil
}