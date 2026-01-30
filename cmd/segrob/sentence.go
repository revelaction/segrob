package main

import (
	"fmt"
	"strconv"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func sentenceCommand(opts SentenceOptions, argDoc string, sentId int, isFilesystem bool, ui UI) error {
	repoPath := opts.DocPath
	if repoPath == "" {
		repoPath = argDoc
	}

	var repo storage.DocRepository
	if !isFilesystem {
		pool, err := zombiezen.NewPool(repoPath)
		if err != nil {
			return err
		}
		defer pool.Close()
		repo = zombiezen.NewDocStore(pool)
	} else {
		h, err := filesystem.NewDocStore(repoPath)
		if err != nil {
			return err
		}
		repo = h
	}

	docId := 0
	if opts.DocPath != "" {
		docId, _ = strconv.Atoi(argDoc)
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
