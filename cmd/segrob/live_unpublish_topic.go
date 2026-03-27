package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

// liveUnpublishTopicCommand removes a topic from the live topics repository.
// The command is idempotent: a DELETE is issued; if the topic doesn't exist,
// that's fine (already removed).
func liveUnpublishTopicCommand(topicDeleter storage.TopicDeleter, opts LiveUnpublishTopicOptions, name string, ui UI) error {
	err := topicDeleter.Delete(name)
	if err != nil {
		// For SQLite backend, DELETE with WHERE clause returns no error if row doesn't exist
		// So any error here is a real failure
		return fmt.Errorf("failed to delete topic %s: %w", name, err)
	}

	_, _ = fmt.Fprintf(ui.Err, "Topic %s removed.\n", name)
	return nil
}
