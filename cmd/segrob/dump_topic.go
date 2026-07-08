package main

import (
	"fmt"
	"sort"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

// dumpTopics reads all topics for the given userID from src, deduplicates each
// topic's expressions, sorts the topics alphabetically by name, and writes the
// result as indented JSON to ui.Out.
//
// Deduplication: each topic may contain duplicate TopicExpr entries (structurally
// identical items in the same order). These are removed so every expression
// appears only once per topic.
//
// Ordering: topics are sorted ascending by Name. Within each topic, expressions
// retain their stored order after deduplication.
//
// Output example:
//
//	[
//	  {
//	    "name": "topic-alpha",
//	    "exprs": [
//	      {"items": [{"lemma": "world", "pos": "NOUN"}]},
//	      {"items": [{"tag": "NNP"}], "flagged": true}
//	    ]
//	  },
//	  {
//	    "name": "topic-beta",
//	    "exprs": [
//	      {"items": [{"near": 2, "lemma": "hello"}]}
//	    ]
//	  }
//	]
func dumpTopics(src storage.TopicReader, userID string, ui UI) error {
	topics, err := src.ReadAll(userID)
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
