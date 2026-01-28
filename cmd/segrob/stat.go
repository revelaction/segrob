package main

import (
	"fmt"
	"strconv"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
	"github.com/revelaction/segrob/storage/filesystem"
)

func statCommand(source string, sentId *int, isFile bool, ui UI) error {
	if isFile {
		return statFile(source, sentId, ui)
	}

	id, err := strconv.Atoi(source)
	if err != nil {
		return fmt.Errorf("invalid DB ID: %v", err)
	}
	return statDocDB(id, sentId, ui)
}

func statFile(path string, sentId *int, ui UI) error {
	doc, err := filesystem.ReadDoc(path)
	if err != nil {
		return err
	}

	if sentId != nil {
		if *sentId < 0 || *sentId >= len(doc.Tokens) {
			return fmt.Errorf("sentence index %d out of bounds (doc has %d sentences)", *sentId, len(doc.Tokens))
		}
		doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*sentId]}}
	}

	hdl := stat.NewHandler()
	hdl.Aggregate(doc)

	stats := hdl.Get()
	fmt.Fprintf(ui.Out, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)

	return nil
}

func statDocDB(docId int, sentId *int, ui UI) error {
	return fmt.Errorf("database mode not implemented")
}
