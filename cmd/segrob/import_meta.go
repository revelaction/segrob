package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func importMetaCommand(corpusRepo storage.CorpusRepository, docRepo storage.DocRepository, opts ImportMetaOptions, ui UI) error {

	// Read metadata from corpus
	meta, err := corpusRepo.ReadMeta(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read corpus meta for %s: %w", opts.ID, err)
	}

	// Check if id already exists in segrob.db (idempotent)
	exists, err := docRepo.Exists(meta.ID)
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
	if _, err := docRepo.WriteMeta(meta.ID, meta.Epub, labels); err != nil {
		return fmt.Errorf("failed to write meta for %s: %w", meta.ID, err)
	}

	// Success: print to stderr
	fmt.Fprintf(ui.Err, "%s %s %s\n", meta.ID, meta.Epub, meta.Labels)

	return nil
}
