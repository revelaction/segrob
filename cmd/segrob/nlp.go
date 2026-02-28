package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

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

	// Check if already processed to avoid duplication
	hasNLP, err := storeTo.HasSentences(opts.ID)
	if err != nil {
		return err
	}
	if hasNLP {
		fmt.Fprintf(ui.Err, "Document %s already has NLP data, skipping.\n", opts.ID)
		return nil
	}

	// Read raw text
	txtBytes, err := storeFrom.ReadTxt(opts.ID)
	if err != nil {
		return err
	}

	// Setup nlp process buffer and execution
	// f ex: python scripts/nlp.py  -> exec.Command("python", "scripts/nlp.py", "-")
	parts := strings.Fields(opts.NlpScript)
	if len(parts) == 0 {
		return fmt.Errorf("NLP script command is empty")
	}

	cmdArgs := append(parts[1:], "-") // from stdin
	cmd := exec.Command(parts[0], cmdArgs...)

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
