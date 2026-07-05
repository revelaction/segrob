//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/revelaction/segrob/topic"
)

// oldTopicExpr mirrors the legacy slice-alias format for unmarshalling only.
type oldTopicExpr []topic.TopicExprItem

func main() {
	inDir := "corpus/topic"
	outDir := "corpus/topic_v2"

	if err := os.MkdirAll(outDir, 0755); err != nil {
		panic(err)
	}

	files, err := os.ReadDir(inDir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		inPath := filepath.Join(inDir, file.Name())
		outPath := filepath.Join(outDir, file.Name())

		b, err := os.ReadFile(inPath)
		if err != nil {
			panic(err)
		}

		var oldExprs []oldTopicExpr
		if err := json.Unmarshal(b, &oldExprs); err != nil {
			fmt.Printf("Skipping %s (already migrated or invalid?)\n", inPath)
			continue
		}

		exprs := make([]topic.TopicExpr, 0, len(oldExprs))
		for _, oe := range oldExprs {
			exprs = append(exprs, topic.TopicExpr{Items: oe, Flagged: false})
		}

		exprs = topic.Deduplicate(exprs)

		out, err := json.MarshalIndent(exprs, "", "  ")
		if err != nil {
			panic(err)
		}

		if err := os.WriteFile(outPath, out, 0644); err != nil {
			panic(err)
		}
		fmt.Printf("Migrated %s to %s\n", inPath, outPath)
	}
}
