package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

func liveShowSentCommand(repo storage.DocRepository, opts LiveShowSentOptions, docId string, sentId int, ui UI) error {
	zero := 0
	sentences, err := repo.Nlp(docId, sentId, &zero)
	if err != nil {
		return err
	}

	if len(sentences) == 0 {
		return fmt.Errorf("sentence index %d not found", sentId)
	}

	if opts.Stats {
		printStats(sentences, ui)
		return nil
	}

	s := sentences[0]
	r := render.NewCLIRenderer()
	r.HasColor = false
	prefix := fmt.Sprintf("✍  %d ", sentId)
	r.Sentence(s.Tokens, prefix)
	_, err = fmt.Fprintln(ui.Out)
	if err != nil {
		return err
	}

	for _, token := range s.Tokens {
		_, err = fmt.Fprintf(ui.Out, "%20q %15q %8s %6d %6d %8s %s\n", token.Text, token.Lemma, token.Pos, token.Id, token.Head, token.Dep, token.Tag)
		if err != nil {
			return err
		}
	}

	return nil
}
