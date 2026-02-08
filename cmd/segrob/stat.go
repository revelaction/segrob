package main

import (
	"fmt"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
	"github.com/revelaction/segrob/storage"
)

func statCommand(repo storage.DocRepository, opts StatOptions, docId int, sentId *int, ui UI) error {
	doc, err := repo.Read(docId)
	if err != nil {
		return err
	}

	if sentId != nil {
		if *sentId < 0 || *sentId >= len(doc.Sentences) {
			return fmt.Errorf("sentence index %d out of bounds (doc has %d sentences)", *sentId, len(doc.Sentences))
		}
		doc = sent.Doc{Sentences: []sent.Sentence{doc.Sentences[*sentId]}}
	}

	hdl := stat.NewHandler()
	hdl.Aggregate(doc)

	stats := hdl.Get()
	fmt.Fprintf(ui.Out, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)

	return nil
}