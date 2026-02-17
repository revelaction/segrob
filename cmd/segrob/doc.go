package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func docCommand(repo storage.DocRepository, opts DocOptions, id int, ui UI) error {
	if p, ok := repo.(storage.Preloader); ok {
		if err := p.LoadNLP(nil, &id, nil); err != nil {
			return err
		}
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
	if start >= len(doc.Sentences) {
		return
	}

	sentences := doc.Sentences[start:]
	if opts.Count != nil {
		limit := *opts.Count
		if limit < len(sentences) {
			sentences = sentences[:limit]
		}
	}

	r := render.NewCLIRenderer()
	r.HasColor = false
	for i, sentence := range sentences {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence.Tokens, prefix)
	}
}
