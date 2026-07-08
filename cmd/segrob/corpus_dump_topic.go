package main

import (
	"github.com/revelaction/segrob/storage"
)

func corpusDumpTopicCommand(src storage.TopicReader, opts CorpusDumpTopicOptions, ui UI) error {
	return dumpTopics(src, "", ui)
}
