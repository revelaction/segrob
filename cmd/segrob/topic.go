package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

// liveLsTopicCommand lists all topics
func liveLsTopicCommand(tr storage.TopicRepository, opts LiveLsTopicOptions, ui UI) error {

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	for topicId, name := range topicLib.Names() {
		_, _ = fmt.Fprintf(ui.Out, "📖 %d %s \n", topicId, name)
	}

	return nil
}
