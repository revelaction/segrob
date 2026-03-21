package main

import (
	"fmt"
	"github.com/revelaction/segrob/storage"
)

func corpusLsTopicCommand(tr storage.TopicRepository, opts CorpusLsTopicOptions, ui UI) error {
	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	for topicId, name := range topicLib.Names() {
		fmt.Fprintf(ui.Out, "📖 %d %s \n", topicId, name)
	}
	return nil
}
