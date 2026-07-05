package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

func corpusIngestTopicCommand(dst storage.TopicWriter, opts CorpusIngestTopicOptions, ui UI) error {
	data, err := os.ReadFile(opts.File)
	if err != nil {
		return fmt.Errorf("failed to read topics file %s: %w", opts.File, err)
	}

	var topics []tpc.Topic
	if err := json.Unmarshal(data, &topics); err != nil {
		return fmt.Errorf("failed to parse topics from %s: %w", opts.File, err)
	}

	if len(topics) == 0 {
		_, _ = fmt.Fprintf(ui.Err, "No topics found in %s.\n", opts.File)
		return nil
	}

	for _, tp := range topics {
		tp.Exprs = tpc.Deduplicate(tp.Exprs)
		_, err = dst.Upsert("", tp, nil)
		if err != nil {
			return fmt.Errorf("failed to ingest topic %s: %w", tp.Name, err)
		}
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully ingested %d topics from %s\n", len(topics), opts.File)
	return nil
}
