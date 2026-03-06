package main

import (
	"encoding/json"
	"fmt"

	"github.com/revelaction/segrob/storage"
)

type nlpSentence struct {
	ID     int             `json:"id"`
	Tokens json.RawMessage `json:"tokens"`
}

type nlpPayload struct {
	Sentences []nlpSentence `json:"sentences"`
}

func catNlpCommand(repo storage.CorpusRepository, opts CatNlpOptions, ui UI) error {
	nlpData, err := repo.ReadNlp(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read nlp for %s: %w", opts.ID, err)
	}

	if opts.NoLemmas {
		var payload nlpPayload
		if err := json.Unmarshal(nlpData, &payload); err != nil {
			return fmt.Errorf("failed to parse nlp json: %w", err)
		}

		nlpData, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal modified nlp json: %w", err)
		}
	}

	// stdout: write bytes directly
	if _, err := ui.Out.Write(nlpData); err != nil {
		return err
	}
	_, err = ui.Out.Write([]byte("\n"))
	return err
}
