package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func importMetaCommand(opts ImportMetaOptions, ui UI) error {
	// Open corpus database (read-only source)
	corpusPool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return fmt.Errorf("failed to open corpus database: %w", err)
	}
	defer corpusPool.Close()

	// Open segrob database (write target)
	docPool, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return fmt.Errorf("failed to open segrob database: %w", err)
	}
	defer docPool.Close()

	corpusStore := zombiezen.NewCorpusStore(corpusPool)
	docStore := zombiezen.NewDocStore(docPool)

	// Read metadata from corpus
	meta, err := corpusStore.ReadMeta(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read corpus meta for %s: %w", opts.ID, err)
	}

	// Check if id already exists in segrob.db (idempotent)
	exists, err := docStore.Exists(meta.ID)
	if err != nil {
		return fmt.Errorf("failed to check existence for %s: %w", meta.ID, err)
	}
	if exists {
		return nil
	}

	// Split comma-separated labels into slice
	var labels []string
	if meta.Labels != "" {
		labels = strings.Split(meta.Labels, ",")
	}

	// Write to segrob.db; source is the epub basename
	if err := docStore.WriteMeta(meta.ID, meta.Epub, labels); err != nil {
		return fmt.Errorf("failed to write meta for %s: %w", meta.ID, err)
	}

	// Success: print to stderr
	fmt.Fprintf(ui.Err, "%s %s %s\n", meta.ID, meta.Epub, meta.Labels)

	return nil
}
