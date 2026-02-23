package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	sent "github.com/revelaction/segrob/sentence"
)

func importNlpCommand(p *Pool, opts ImportNlpOptions, ui UI) error {
	var input io.Reader
	if opts.From != "" {
		f, err := os.Open(opts.From)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer f.Close()
		input = f
	} else {
		input = os.Stdin
	}

	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var doc struct {
		Sentences []sent.SentenceIngest `json:"sentences"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	repo, err := NewDocRepository(p, opts.DbPath)
	if err != nil {
		return err
	}

	if err := repo.WriteNLP(opts.DocID, doc.Sentences); err != nil {
		return fmt.Errorf("failed to write NLP data: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully imported %d sentences for doc ID %d\n", len(doc.Sentences), opts.DocID)
	return nil
}
