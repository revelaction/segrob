//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/revelaction/segrob/topic"
)

type oldTopicExpr []topic.TopicExprItem

func main() {
	inDir := "corpus/topic"
	outFile := "corpus/topics.json"

	files, err := os.ReadDir(inDir)
	if err != nil {
		panic(err)
	}

	var topics []topic.Topic
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		inPath := filepath.Join(inDir, file.Name())

		b, err := os.ReadFile(inPath)
		if err != nil {
			panic(err)
		}

		var oldExprs []oldTopicExpr
		if err := json.Unmarshal(b, &oldExprs); err != nil {
			fmt.Printf("Skipping %s (invalid format?)\n", inPath)
			continue
		}

		exprs := make([]topic.TopicExpr, 0, len(oldExprs))
		for _, oe := range oldExprs {
			exprs = append(exprs, topic.TopicExpr{Items: []topic.TopicExprItem(oe), Flagged: false})
		}

		exprs = topic.Deduplicate(exprs)

		name := file.Name()[:len(file.Name())-len(filepath.Ext(file.Name()))]
		topics = append(topics, topic.Topic{Name: name, Exprs: exprs})
	}

	sort.Slice(topics, func(i, j int) bool { return topics[i].Name < topics[j].Name })

	out, err := topic.Library(topics).MarshalIndent()
	if err != nil {
		panic(err)
	}

	if err := os.WriteFile(outFile, out, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("Migrated %d topics from %s to %s\n", len(topics), inDir, outFile)
}
