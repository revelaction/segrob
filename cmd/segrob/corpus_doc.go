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

func corpusShowCommand(repo storage.CorpusRepository, opts ShowOptions, id string, ui UI) error {
	nlpData, err := repo.ReadNlp(id)
	if err != nil {
		return fmt.Errorf("failed to read nlp for %s: %w", id, err)
	}

	if len(nlpData) == 0 {
		fmt.Fprintf(ui.Out, "no nlp payload for %s\n", id)
		return nil
	}

	var payload nlpResponse
	if err := json.Unmarshal(nlpData, &payload); err != nil {
		return fmt.Errorf("failed to parse nlp json: %w", err)
	}

	renderDoc(payload.Sentences, opts, ui)
	return nil
}
