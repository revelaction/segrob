package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func docCommand(repo storage.DocRepository, opts DocOptions, id int, ui UI) error {
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
