package main

import (
	"fmt"
	"strconv"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage/filesystem"
)

func docCommand(opts DocOptions, arg string, isFile bool, ui UI) error {
	if isFile {
		return renderFile(arg, opts, ui)
	}

	if arg != "" {
		id, _ := strconv.Atoi(arg)
		return renderDocDB(id, opts, ui)
	}

	return listDocsDB(ui)
}

func renderFile(path string, opts DocOptions, ui UI) error {
	doc, err := filesystem.ReadDoc(path)
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
	if start >= len(doc.Tokens) {
		return
	}

	sentences := doc.Tokens[start:]
	if opts.Count >= 0 && opts.Count < len(sentences) {
		sentences = sentences[:opts.Count]
	}

	r := render.NewRenderer()
	r.HasColor = false
	for i, sentence := range sentences {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence, prefix)
	}
}

func listDocsDB(ui UI) error {
	return fmt.Errorf("database mode not implemented")
}

func renderDocDB(id int, opts DocOptions, ui UI) error {
	return fmt.Errorf("database mode not implemented")
}
