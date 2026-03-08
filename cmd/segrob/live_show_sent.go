package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

func liveShowSentCommand(repo storage.DocRepository, opts LiveShowSentOptions, docId string, sentId int, ui UI) error {
	sentences, err := repo.Nlp(docId)
	if err != nil {
		return err
	}

	if sentId < 0 || sentId >= len(sentences) {
		return fmt.Errorf("sentence index %d out of bounds (0-%d)", sentId, len(sentences)-1)
	}

	if opts.Stats {
		printStats(sentences[sentId:sentId+1], ui)
		return nil
	}

	s := sentences[sentId]
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
