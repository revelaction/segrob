package main

import (
	"fmt"
	"strconv"

	"github.com/revelaction/segrob/file"
	"github.com/revelaction/segrob/render"
)

func sentenceCommand(source string, sentId int, isFile bool, ui UI) error {
	if isFile {
		return renderSentenceFile(source, sentId, ui)
	}

	docId, _ := strconv.Atoi(source)
	return renderSentenceDB(docId, sentId, ui)
}

func renderSentenceFile(path string, sentId int, ui UI) error {
	doc, err := file.ReadDoc(path)
	if err != nil {
		return err
	}

	if sentId < 0 || sentId >= len(doc.Tokens) {
		return fmt.Errorf("sentence index %d out of bounds (0-%d)", sentId, len(doc.Tokens)-1)
	}

	s := doc.Tokens[sentId]
	r := render.NewRenderer()
	r.HasColor = false
	prefix := fmt.Sprintf("✍  %d ", sentId)
	r.Sentence(s, prefix)
	fmt.Fprintln(ui.Out)

	for _, token := range s {
		fmt.Fprintf(ui.Out, "%20q %15q %8s %6d %6d %8s %s\n", token.Text, token.Lemma, token.Pos, token.Id, token.Head, token.Dep, token.Tag)
	}

	return nil
}

func renderSentenceDB(docId int, sentId int, ui UI) error {
	return fmt.Errorf("database mode not implemented")
}

