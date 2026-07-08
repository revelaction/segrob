package main

import (
	"github.com/revelaction/segrob/storage"
)

func liveDumpTopicCommand(src storage.TopicReader, opts LiveDumpTopicOptions, ui UI) error {
	return dumpTopics(src, opts.UserID, ui)
}
