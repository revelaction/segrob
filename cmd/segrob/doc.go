package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func docCommand(repo storage.DocRepository, opts DocOptions, id int, ui UI) error {
	sentences, err := repo.Nlp(id)
	if err != nil {
		return err
	}

	renderDoc(sentences, opts, ui)
	return nil
}

func renderDoc(sentences []sent.Sentence, opts DocOptions, ui UI) {
	start := opts.Start
	if start < 0 {
		start = 0
	}
	if start >= len(sentences) {
		return
	}

	subset := sentences[start:]
	if opts.Count != nil {
		limit := *opts.Count
		if limit < len(subset) {
			subset = subset[:limit]
		}
	}

	r := render.NewCLIRenderer()
	r.HasColor = false
	for i, sentence := range subset {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence.Tokens, prefix)
	}
}
