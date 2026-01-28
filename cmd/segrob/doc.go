package main

import (
	"fmt"
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
		repo = zombiezen.NewDocHandler(pool)
	} else {
		h, err := filesystem.NewDocHandler(opts.DocPath)
		if err != nil {
			return err
		}
		if arg == "" {
			if err := h.LoadNames(); err != nil {
				return err
			}
		} else {
			if err := h.LoadNames(); err != nil {
				return err
			}
			if err := h.LoadContents(nil); err != nil {
				return err
			}
		}
		repo = h
	}

	if arg == "" {
		return listDocs(repo, ui)
	}

	id, _ := strconv.Atoi(arg)
	doc, err := repo.Doc(id)
	if err != nil {
		return err
	}

	renderDoc(doc, opts, ui)
	return nil
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

func listDocs(repo storage.DocReader, ui UI) error {
	names, err := repo.Names()
	if err != nil {
		return err
	}

	for i, name := range names {
		fmt.Fprintf(ui.Out, "📖 %d %s\n", i, name)
	}
	return nil
}
