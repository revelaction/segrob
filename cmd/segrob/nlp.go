package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func nlpCommand(opts NlpOptions, ui UI) error {
	// Pool From (corpus)
	poolFrom, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return fmt.Errorf("failed to open corpus database: %w", err)
	}
	defer poolFrom.Close()
	storeFrom := zombiezen.NewCorpusStore(poolFrom)

	// Pool To (segrob)
	poolTo, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return fmt.Errorf("failed to open segrob database: %w", err)
	}
	defer poolTo.Close()
	storeTo := zombiezen.NewDocStore(poolTo)

	// Read raw text
	txtBytes, err := storeFrom.ReadTxt(opts.ID)
	if err != nil {
		return err
	}

	// Setup python process buffer and execution
	cmd := exec.Command(opts.NlpScript, "-")
	cmd.Stdin = bytes.NewReader(txtBytes)

	// We read everything directly into memory
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = ui.Err // redirect python stderr to user UI

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("NLP script failed: %w", err)
	}

	// Decode the JSON single payload
	var doc struct {
		Sentences []storage.SentenceIngest `json:"sentences"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		return fmt.Errorf("failed to unmarshal NLP JSON payload: %w", err)
	}

	// Write to database via the pre-existing buffer method
	if err := storeTo.WriteNLP(opts.ID, doc.Sentences); err != nil {
		return fmt.Errorf("failed to write NLP data: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully imported %d sentences for doc ID %s\n", len(doc.Sentences), opts.ID)
	return nil
}
