package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/revelaction/segrob/storage"
)

// corpusPublishLabelCommand pushes the labels stored in the corpus table for
// the given document into the live tables. It runs three short transactions:
//
//  1. Delete the label index (sentence_labels) — stale data disappears immediately.
//  2. Update the docs row and upsert into the labels table.
//  3. Rebuild the label index from the fresh label IDs.
//
// The command is idempotent: re-running produces the same final state.
func corpusPublishLabelCommand(corpusRepo storage.CorpusRepository, docRepo storage.DocRepository, opts CorpusPublishLabelOptions, ui UI) error {
	id := opts.ID

	// Verify the document exists in the live tables.
	exists, err := docRepo.Exists(id)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		fmt.Fprintf(ui.Err, "Document %s not found in live tables (publish it first).\n", id)
		return nil
	}

	// Read the current labels from the corpus.
	meta, err := corpusRepo.ReadMeta(id)
	if err != nil {
		return fmt.Errorf("failed to read corpus meta for %s: %w", id, err)
	}

	var labels []string
	if meta.Labels != "" {
		labels = strings.Split(meta.Labels, ",")
	}

	// Transaction 1 — cut the label index (live switch for label filtering).
	start := time.Now()
	if err := docRepo.DeleteLabelsOptimization(id); err != nil {
		fmt.Fprintf(ui.Err, "DeleteLabelsOpt ❌ %v\n", err)
		return fmt.Errorf("DeleteLabelsOptimization failed: %w", err)
	}
	fmt.Fprintf(ui.Err, "DeleteLabelsOpt ✅ %s\n", time.Since(start))

	// Transaction 2 — upsert labels table and update docs row.
	// Note: We do not clean up potentially orphaned labels in the labels table.
	start = time.Now()
	labelIDs, err := docRepo.UpdateLabels(id, labels)
	if err != nil {
		fmt.Fprintf(ui.Err, "UpdateLabels    ❌ %v\n", err)
		return fmt.Errorf("UpdateLabels failed: %w", err)
	}
	fmt.Fprintf(ui.Err, "UpdateLabels    ✅ %s\n", time.Since(start))

	// Transaction 3 — rebuild the label index.
	start = time.Now()
	if err := docRepo.WriteLabelsOptimization(id, labelIDs); err != nil {
		fmt.Fprintf(ui.Err, "WriteLabelsOpt  ❌ %v\n", err)
		return fmt.Errorf("WriteLabelsOptimization failed: %w", err)
	}
	fmt.Fprintf(ui.Err, "WriteLabelsOpt  ✅ %s\n", time.Since(start))

	fmt.Fprintf(ui.Err, "\nLabels published for %s.\n", id)
	return nil
}
