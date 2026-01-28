package main

import (
	"fmt"
	"strconv"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func docCommand(opts DocOptions, arg string, isArgFile bool, isRepoFile bool, ui UI) error {
	if isArgFile {
		return renderFile(arg, opts, ui)
	}

	if arg != "" {
		id, _ := strconv.Atoi(arg)
		return renderDocDB(id, opts, isRepoFile, ui)
	}

	return listDocsDB(opts, isRepoFile, ui)
}

func renderFile(path string, opts DocOptions, ui UI) error {
	doc, err := filesystem.ReadDoc(path)
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

func listDocsDB(opts DocOptions, isRepoFile bool, ui UI) error {
	var names []string
	var err error

	if isRepoFile {
		pool, err := zombiezen.NewPool(opts.DocPath)
		if err != nil {
			return err
		}
		defer pool.Close()
		repo := zombiezen.NewDocHandler(pool)
		names, err = repo.Names()
	} else {
		repo, err := filesystem.NewDocHandler(opts.DocPath)
		if err != nil {
			return err
		}
		if err := repo.Load(nil); err != nil {
			return err
		}
		names, err = repo.Names()
	}

	if err != nil {
		return err
	}

	for _, name := range names {
		fmt.Fprintln(ui.Out, name)
	}
	return nil
}

func renderDocDB(id int, opts DocOptions, isRepoFile bool, ui UI) error {
	var doc sent.Doc
	var err error

	if isRepoFile {
		pool, err := zombiezen.NewPool(opts.DocPath)
		if err != nil {
			return err
		}
		defer pool.Close()
		repo := zombiezen.NewDocHandler(pool)
		doc, err = repo.Doc(id)
	} else {
		repo, err := filesystem.NewDocHandler(opts.DocPath)
		if err != nil {
			return err
		}
		if err := repo.Load(nil); err != nil {
			return err
		}
		names, err := repo.Names()
		if err != nil {
			return err
		}
		if id < 0 || id >= len(names) {
			return fmt.Errorf("invalid doc index: %d", id)
		}
		doc, err = repo.DocForName(names[id])
	}

	if err != nil {
		return err
	}

	renderDoc(doc, opts, ui)
	return nil
}
