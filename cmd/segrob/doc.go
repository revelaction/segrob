package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
)

var digitRegex = regexp.MustCompile(`^\d+$`)

func docCommand(opts DocOptions, arg string, ui UI) error {
	if arg == "" {
		return listDocsDB(ui)
	}

	// 1. Check if it is a file
	if info, err := os.Stat(arg); err == nil && !info.IsDir() {
		return renderFile(arg, opts, ui)
	}

	// 2. Check if it is an integer (DB ID)
	if digitRegex.MatchString(arg) {
		id, _ := strconv.Atoi(arg)
		return renderDocDB(id, opts, ui)
	}

	return fmt.Errorf("file not found and not a valid DB ID: %s", arg)
}

func renderFile(path string, opts DocOptions, ui UI) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var doc sent.Doc
	if err := json.Unmarshal(content, &doc); err != nil {
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
