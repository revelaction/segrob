package main

import (
	"fmt"
	"time"

	"github.com/revelaction/segrob/storage"
)

// liveUnpublishCommand removes a document from all live tables in the reverse
// order of publish. The lemma index (live switch) is cut first so the document
// disappears from FindCandidates immediately; the remaining phases clean up the
// supporting rows. Each phase is idempotent: if the data is already gone it
// prints "(already removed)" and continues.
func liveUnpublishCommand(docRepo storage.DocRepository, opts LiveUnpublishOptions, ui UI) error {
	id := opts.ID

	// Verify the document exists before starting.
	exists, err := docRepo.Exists(id)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		_, _ = fmt.Fprintf(ui.Err, "Document %s not found in live tables.\n", id)
		return nil
	}

	// Phase 1 — THE LIVE SWITCH: cut lemma index first.
	hasLemmas, err := docRepo.HasLemmaOptimization(id)
	if err != nil {
		return fmt.Errorf("failed to check lemma optimization: %w", err)
	}
	if hasLemmas {
		start := time.Now()
		dErr := docRepo.DeleteLemmaOptimization(id)
		if dErr != nil {
			_, _ = fmt.Fprintf(ui.Err, "DeleteLemmaOpt  ❌ %v\n", dErr)
			return fmt.Errorf("DeleteLemmaOptimization failed: %w", dErr)
		}
		_, _ = fmt.Fprintf(ui.Err, "DeleteLemmaOpt  ✅ %s\n", time.Since(start))
	} else {
		_, _ = fmt.Fprintf(ui.Err, "DeleteLemmaOpt  ✅ (already removed)\n")
	}

	// Phase 2 — remove label index.
	hasLabels, err := docRepo.HasLabelsOptimization(id)
	if err != nil {
		return fmt.Errorf("failed to check labels optimization: %w", err)
	}
	if hasLabels {
		start := time.Now()
		dErr := docRepo.DeleteLabelsOptimization(id)
		if dErr != nil {
			_, _ = fmt.Fprintf(ui.Err, "DeleteLabelsOpt ❌ %v\n", dErr)
			return fmt.Errorf("DeleteLabelsOptimization failed: %w", dErr)
		}
		_, _ = fmt.Fprintf(ui.Err, "DeleteLabelsOpt ✅ %s\n", time.Since(start))
	} else {
		_, _ = fmt.Fprintf(ui.Err, "DeleteLabelsOpt ✅ (already removed)\n")
	}

	// Phase 3 — remove sentences.
	hasSentences, err := docRepo.HasSentences(id)
	if err != nil {
		return fmt.Errorf("failed to check sentences: %w", err)
	}
	if hasSentences {
		start := time.Now()
		dErr := docRepo.DeleteNlpData(id)
		if dErr != nil {
			_, _ = fmt.Fprintf(ui.Err, "DeleteNlpData   ❌ %v\n", dErr)
			return fmt.Errorf("DeleteNlpData failed: %w", dErr)
		}
		_, _ = fmt.Fprintf(ui.Err, "DeleteNlpData   ✅ %s\n", time.Since(start))
	} else {
		_, _ = fmt.Fprintf(ui.Err, "DeleteNlpData   ✅ (already removed)\n")
	}

	// Phase 4 — remove doc row.
	exists, err = docRepo.Exists(id)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if exists {
		start := time.Now()
		dErr := docRepo.DeleteMeta(id)
		if dErr != nil {
			_, _ = fmt.Fprintf(ui.Err, "DeleteMeta      ❌ %v\n", dErr)
			return fmt.Errorf("DeleteMeta failed: %w", dErr)
		}
		_, _ = fmt.Fprintf(ui.Err, "DeleteMeta      ✅ %s\n", time.Since(start))
	} else {
		_, _ = fmt.Fprintf(ui.Err, "DeleteMeta      ✅ (already removed)\n")
	}

	_, _ = fmt.Fprintf(ui.Err, "\nUnpublished %s.\n", id)
	return nil
}
