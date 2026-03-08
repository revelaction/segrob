package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/revelaction/segrob/storage"
)

func corpusPublishCommand(corpusRepo storage.CorpusRepository, docRepo storage.DocRepository, opts CorpusPublishOptions, ui UI) error {
	// Read NLP data from corpus
	nlpBytes, err := corpusRepo.ReadNlp(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read NLP data for %s: %w", opts.ID, err)
	}
	if len(nlpBytes) == 0 {
		return fmt.Errorf("no NLP data found for %s in corpus", opts.ID)
	}

	// Decode the JSON envelope (tokens stay as json.RawMessage)
	var doc struct {
		Sentences []storage.SentenceIngest `json:"sentences"`
	}
	if err := json.Unmarshal(nlpBytes, &doc); err != nil {
		return fmt.Errorf("failed to unmarshal NLP JSON: %w", err)
	}

	// Read metadata from corpus for WriteMeta
	meta, err := corpusRepo.ReadMeta(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read corpus meta for %s: %w", opts.ID, err)
	}

	var labels []string
	var labelIDs []int

	if meta.Labels != "" {
		labels = strings.Split(meta.Labels, ",")
	}

	// Transaction 1: WriteMeta (idempotent — skip if doc already exists)
	exists, err := docRepo.Exists(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if !exists {
		start := time.Now()
		// WriteMeta: upserts labels, INSERT docs with label_ids, returns IDs
		labelIDs, err = docRepo.WriteMeta(meta.ID, meta.Epub, labels) // [3, 7, 12, 15]
		if err != nil {
			fmt.Fprintf(ui.Out, "WriteMeta       ❌ %v\n", err)
			return fmt.Errorf("WriteMeta failed: %w", err)
		}
		fmt.Fprintf(ui.Out, "WriteMeta       ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Out, "WriteMeta       ✅ (already exists)\n")
	}

	// Transaction 2: WriteNlpData (idempotent — skip if sentences exist)
	hasSentences, err := docRepo.HasSentences(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to check sentences: %w", err)
	}
	if !hasSentences {
		start := time.Now()
		if err := docRepo.WriteNlpData(opts.ID, doc.Sentences); err != nil {
			fmt.Fprintf(ui.Out, "WriteNlpData    ❌ %v\n", err)
			return fmt.Errorf("WriteNlpData failed: %w", err)
		}
		fmt.Fprintf(ui.Out, "WriteNlpData    ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Out, "WriteNlpData    ✅ (already exists)\n")
	}

	// Transaction 3: WriteLabelsOptimization (idempotent — skip if labels optimization exists)
	hasLabels, err := docRepo.HasLabelsOptimization(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to check labels optimization: %w", err)
	}
	if !hasLabels {
		start := time.Now()
		if err := docRepo.WriteLabelsOptimization(opts.ID, labelIDs); err != nil {
			fmt.Fprintf(ui.Out, "WriteLabelsOpt  ❌ %v\n", err)
			return fmt.Errorf("WriteLabelsOptimization failed: %w", err)
		}
		fmt.Fprintf(ui.Out, "WriteLabelsOpt  ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Out, "WriteLabelsOpt  ✅ (already exists)\n")
	}

	// Transaction 4: WriteLemmaOptimization — THE LIVE SWITCH (idempotent)
	hasLemmas, err := docRepo.HasLemmaOptimization(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to check lemma optimization: %w", err)
	}
	if !hasLemmas {
		start := time.Now()
		if err := docRepo.WriteLemmaOptimization(opts.ID, doc.Sentences); err != nil {
			fmt.Fprintf(ui.Out, "WriteLemmaOpt   ❌ %v\n", err)
			return fmt.Errorf("WriteLemmaOptimization failed: %w", err)
		}
		fmt.Fprintf(ui.Out, "WriteLemmaOpt   ✅ %s\n", time.Since(start))
	} else {
		fmt.Fprintf(ui.Out, "WriteLemmaOpt   ✅ (already exists)\n")
	}

	// Optional: delete nlp field from corpus
	if opts.Move {
		if err := corpusRepo.ClearNlp(opts.ID); err != nil {
			fmt.Fprintf(ui.Err, "Warning: failed to clear NLP from corpus: %v\n", err)
		}
	}

	return nil
}
