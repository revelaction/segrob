package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

func sentenceCommand(repo storage.DocRepository, opts SentenceOptions, docId int, sentId int, ui UI) error {
	if p, ok := repo.(storage.Preloader); ok {
		if err := p.LoadNLP(nil, &docId, nil); err != nil {
			return err
		}
	}
	doc, err := repo.Read(docId)
	if err != nil {
		return err
	}

	if sentId < 0 || sentId >= len(doc.Sentences) {
		return fmt.Errorf("sentence index %d out of bounds (0-%d)", sentId, len(doc.Sentences)-1)
	}

	s := doc.Sentences[sentId]
	r := render.NewCLIRenderer()
	r.HasColor = false
	prefix := fmt.Sprintf("✍  %d ", sentId)
	r.Sentence(s.Tokens, prefix)
	fmt.Fprintln(ui.Out)

	for _, token := range s.Tokens {
		fmt.Fprintf(ui.Out, "%20q %15q %8s %6d %6d %8s %s\n", token.Text, token.Lemma, token.Pos, token.Id, token.Head, token.Dep, token.Tag)
	}

	return nil
}
