package main

import (
	"fmt"
	"os"

	"github.com/revelaction/segrob/storage"
)

func corpusPushTxtCommand(repo storage.CorpusRepository, opts CorpusPushTxtOptions, ui UI) error {
	txt, err := os.ReadFile(opts.File)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", opts.File, err)
	}

	// Use existing sha256Hex from corpus_ingest_meta.go (same package main)
	txtHash := sha256Hex(txt, 32)

	err = repo.UpdateTxt(opts.ID, txt, txtHash, opts.By, opts.Note)
	if err != nil {
		return fmt.Errorf("failed to update corpus txt for %s: %w", opts.ID, err)
	}

	fmt.Fprintf(ui.Err, "Successfully updated txt for %s from %s\n", opts.ID, opts.File)
	return nil
}
