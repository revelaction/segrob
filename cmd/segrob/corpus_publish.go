package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/revelaction/segrob/storage"
)

// corpusPublishCommand is the single entry point for both single-doc and all-doc modes.
func corpusPublishCommand(corpusRepo storage.CorpusRepository, docRepo storage.DocRepository, opts CorpusPublishOptions, ui UI) error {
	if !opts.All {
		return publishOne(corpusRepo, docRepo, opts.ID, opts.Move, opts.Force, ui)
	}

	metas, err := corpusRepo.List()
	if err != nil {
		return fmt.Errorf("failed to list corpus: %w", err)
	}

	var candidates []storage.CorpusMeta
	for _, m := range metas {
		if m.HasAck() {
			candidates = append(candidates, m)
		}
	}

	if len(candidates) == 0 {
		fmt.Fprintf(ui.Err, "No ACKed documents to publish.\n")
		return nil
	}

	fmt.Fprintf(ui.Err, "Publishing %d ACKed document(s)...\n\n", len(candidates))

	for i, m := range candidates {
		fmt.Fprintf(ui.Err, "[%d/%d] %s\n", i+1, len(candidates), m.ID)
		// force is always false: the HasAck() filter above already guarantees ACK
		if err := publishOne(corpusRepo, docRepo, m.ID, opts.Move, false, ui); err != nil {
			return fmt.Errorf("failed to publish %s: %w\n\nFix the issue and re-run the command to continue", m.ID, err)
		}
		_, err = fmt.Fprintln(ui.Err)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(ui.Err, "Published %d document(s).\n", len(candidates))
	return nil
}

// publishOne publishes a single document through the 4 idempotent transactional phases.
func publishOne(corpusRepo storage.CorpusRepository, docRepo storage.DocRepository, id string, move bool, force bool, ui UI) error {
	// Read NLP data from corpus
	nlpBytes, err := corpusRepo.ReadNlp(id)
	if err != nil {
		return fmt.Errorf("failed to read NLP data for %s: %w", id, err)
	}
	if len(nlpBytes) == 0 {
		return fmt.Errorf("no NLP data found for %s in corpus", id)
	}

	// Decode the JSON envelope (tokens stay as json.RawMessage)
	var doc struct {
		Sentences []storage.SentenceIngest `json:"sentences"`
	}
	if err := json.Unmarshal(nlpBytes, &doc); err != nil {
		return fmt.Errorf("failed to unmarshal NLP JSON: %w", err)
	}

	// Read metadata from corpus for WriteMeta
	meta, err := corpusRepo.ReadMeta(id)
	if err != nil {
		return fmt.Errorf("failed to read corpus meta for %s: %w", id, err)
	}

	if !force {
		if !meta.HasAck() {
			return fmt.Errorf("document %s is not fully acknowledged (use -f/--force to override)", id)
		}
	}

	var labels []string
	var labelIDs []int

	if meta.Labels != "" {
		labels = strings.Split(meta.Labels, ",")
	}

	// Transaction 1: WriteMeta (idempotent — skip if doc already exists)
	exists, err := docRepo.Exists(id)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		start := time.Now()
		// WriteMeta: upserts labels, INSERT docs with label_ids, returns IDs
		labelIDs, err = docRepo.WriteMeta(meta.ID, meta.Epub, labels) // [3, 7, 12, 15]
		if err != nil {
			fmt.Fprintf(ui.Err, "WriteMeta       ❌ %v\n", err)
			return fmt.Errorf("WriteMeta failed: %w", err)
		}
		fmt.Fprintf(ui.Err, "WriteMeta       ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Err, "WriteMeta       ✅ (already exists)\n")
	}

	// Transaction 2: WriteNlpData (idempotent — skip if sentences exist)
	hasSentences, err := docRepo.HasSentences(id)
	if err != nil {
		return fmt.Errorf("failed to check sentences: %w", err)
	}
	if !hasSentences {
		start := time.Now()
		if err := docRepo.WriteNlpData(id, doc.Sentences); err != nil {
			fmt.Fprintf(ui.Err, "WriteNlpData    ❌ %v\n", err)
			return fmt.Errorf("WriteNlpData failed: %w", err)
		}
		fmt.Fprintf(ui.Err, "WriteNlpData    ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Err, "WriteNlpData    ✅ (already exists)\n")
	}

	// Transaction 3: WriteLabelsOptimization (idempotent — skip if labels optimization exists)
	hasLabels, err := docRepo.HasLabelsOptimization(id)
	if err != nil {
		return fmt.Errorf("failed to check labels optimization: %w", err)
	}
	if !hasLabels {
		start := time.Now()
		if err := docRepo.WriteLabelsOptimization(id, labelIDs); err != nil {
			fmt.Fprintf(ui.Err, "WriteLabelsOpt  ❌ %v\n", err)
			return fmt.Errorf("WriteLabelsOptimization failed: %w", err)
		}
		fmt.Fprintf(ui.Err, "WriteLabelsOpt  ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Err, "WriteLabelsOpt  ✅ (already exists)\n")
	}

	// Transaction 4: WriteLemmaOptimization — THE LIVE SWITCH (idempotent)
	hasLemmas, err := docRepo.HasLemmaOptimization(id)
	if err != nil {
		return fmt.Errorf("failed to check lemma optimization: %w", err)
	}
	if !hasLemmas {
		start := time.Now()
		if err := docRepo.WriteLemmaOptimization(id, doc.Sentences); err != nil {
			fmt.Fprintf(ui.Err, "WriteLemmaOpt   ❌ %v\n", err)
			return fmt.Errorf("WriteLemmaOptimization failed: %w", err)
		}
		fmt.Fprintf(ui.Err, "WriteLemmaOpt   ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Err, "WriteLemmaOpt   ✅ (already exists)\n")
	}

	// Optional: delete nlp field from corpus
	if move {
		if err := corpusRepo.ClearNlp(id); err != nil {
			fmt.Fprintf(ui.Err, "Warning: failed to clear NLP from corpus: %v\n", err)
		}
	}

	return nil
}
