package main

import (
	"fmt"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
	"github.com/revelaction/segrob/storage"
)

func statCommand(repo storage.DocRepository, opts StatOptions, docId int, sentId *int, ui UI) error {
	sentences, err := repo.Nlp(docId)
	if err != nil {
		return err
	}

	if sentId != nil {
		if *sentId < 0 || *sentId >= len(sentences) {
			return fmt.Errorf("sentence index %d out of bounds (doc has %d sentences)", *sentId, len(sentences))
		}
		sentences = []sent.Sentence{sentences[*sentId]}
	}

	hdl := stat.NewHandler()
	hdl.Aggregate(sentences)

	stats := hdl.Get()
	fmt.Fprintf(ui.Out, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)

	return nil
}
