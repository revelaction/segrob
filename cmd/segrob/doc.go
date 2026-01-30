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

func docCommand(opts DocOptions, arg string, isFilesystem bool, ui UI) error {
	repoPath := opts.DocPath
	if repoPath == "" {
		repoPath = arg
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

	id := 0
	if opts.DocPath != "" {
		id, _ = strconv.Atoi(arg)
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
