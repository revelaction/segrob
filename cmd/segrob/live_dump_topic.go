package main

import (
	"fmt"
	"sort"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

func liveDumpTopicCommand(src storage.TopicReader, opts LiveDumpTopicOptions, ui UI) error {
	topics, err := src.ReadAll(opts.UserID)
	if err != nil {
		return fmt.Errorf("failed to read topics: %w", err)
	}

	for i := range topics {
		topics[i].Exprs = tpc.Deduplicate(topics[i].Exprs)
	}

	sort.Slice(topics, func(i, j int) bool { return topics[i].Name < topics[j].Name })

	jsonData, err := tpc.Library(topics).MarshalIndent()
	if err != nil {
		return fmt.Errorf("failed to marshal topics: %w", err)
	}

	_, err = ui.Out.Write(jsonData)
	return err
}
