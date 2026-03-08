package main

import (
	"fmt"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
)

func printStats(sentences []sent.Sentence, ui UI) {
	hdl := stat.NewHandler()
	hdl.Aggregate(sentences)

	stats := hdl.Get()
	fmt.Fprintf(ui.Out, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)
}
