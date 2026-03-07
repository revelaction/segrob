package main

import (
	"encoding/json"
	"fmt"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

type nlpResponse struct {
	Sentences []sent.Sentence `json:"sentences"`
}

func corpusDocCommand(repo storage.CorpusRepository, opts CorpusDocOptions, id string, ui UI) error {
	nlpData, err := repo.ReadNlp(id)
	if err != nil {
		return fmt.Errorf("failed to read nlp for %s: %w", id, err)
	}

	var payload nlpResponse
	if err := json.Unmarshal(nlpData, &payload); err != nil {
		return fmt.Errorf("failed to parse nlp json: %w", err)
	}

	docOpts := DocOptions{
		Start: opts.Start,
		Count: opts.Count,
	}

	// This reuses renderDoc from cmd/segrob/render_doc.go
	renderDoc(payload.Sentences, docOpts, ui)
	return nil
}
