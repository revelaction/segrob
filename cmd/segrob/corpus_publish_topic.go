package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func corpusPublishTopicCommand(
	corpusTopics storage.TopicReader,
	liveTopics storage.TopicWriter,
	opts CorpusPublishTopicOptions,
	ui UI,
) error {
	// 1. Read all topics from corpus
	topics, readErr := corpusTopics.ReadAll()
	if readErr != nil {
		return fmt.Errorf("failed to read topics from corpus: %w", readErr)
	}

	if len(topics) == 0 {
		_, printErr := fmt.Fprintf(ui.Err, "No topics to publish.\n")
		return printErr
	}

	_, printErr := fmt.Fprintf(ui.Err, "Publishing %d topic(s) from corpus to live...\n", len(topics))
	if printErr != nil {
		return printErr
	}

	// 2. Iterate and write to live (idempotent upsert logic built-in to Write)
	publishedCount := 0
	for _, tp := range topics {
		writeErr := liveTopics.Write(tp)
		if writeErr != nil {
			return fmt.Errorf("failed to write topic %q to live: %w", tp.Name, writeErr)
		}
		publishedCount++
	}

	// 3. Output result
	_, printErr = fmt.Fprintf(ui.Err, "✅ Successfully published %d topic(s).\n", publishedCount)
	return printErr
}
