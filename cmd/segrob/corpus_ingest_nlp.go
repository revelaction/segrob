package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func corpusIngestNlpCommand(corpusRepo storage.CorpusRepository, opts CorpusIngestNlpOptions, ui UI) error {
	// Check TxtAck status unless forced
	if !opts.Force {
		meta, err := corpusRepo.ReadMeta(opts.ID)
		if err != nil {
			return fmt.Errorf("failed to read corpus meta: %w", err)
		}
		if !meta.TxtAck {
			return fmt.Errorf("text not acknowledged for doc ID %s (use -f/--force to override)", opts.ID)
		}
	}

	// Read raw text
	txtBytes, err := corpusRepo.ReadTxt(opts.ID)
	if err != nil {
		return err
	}

	// Execute NLP script
	parts := strings.Fields(opts.NlpScript)
	if len(parts) == 0 {
		return fmt.Errorf("NLP script command is empty")
	}
	cmdArgs := append(parts[1:], "-")
	cmd := exec.Command(parts[0], cmdArgs...)
	cmd.Stdin = bytes.NewReader(txtBytes)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = ui.Err

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NLP script failed: %w", err)
	}

	// Validate JSON output (O(N) time, 0 allocations)
	// This catches trailing garbage like "{} -" or leaked debug logs.
	if !json.Valid(out.Bytes()) {
		return fmt.Errorf("NLP script produced invalid JSON output (check for leaked logs or extra arguments)")
	}

	// Store raw JSON in corpus.nlp
	if err := corpusRepo.WriteNlp(opts.ID, out.Bytes()); err != nil {
		return fmt.Errorf("failed to write NLP data to corpus: %w", err)
	}

	fmt.Fprintf(ui.Err, "NLP data stored for doc ID %s (%d bytes)\n", opts.ID, out.Len())
	return nil
}
