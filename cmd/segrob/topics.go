package main

import (
	"fmt"
	"strconv"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage/filesystem"
)

func topicsCommand(opts TopicsOptions, source string, sentId int, isTopicFile, isSourceFile bool, ui UI) error {
	if isSourceFile {
		return topicsFile(source, sentId, isTopicFile, opts, ui)
	}

	id, err := strconv.Atoi(source)
	if err != nil {
		return fmt.Errorf("invalid DB ID: %v", err)
	}
	return topicsDocDB(id, sentId, isTopicFile, opts, ui)
}

func topicsFile(path string, sentId int, isTopicFile bool, opts TopicsOptions, ui UI) error {
	doc, err := filesystem.ReadDoc(path)
	if err != nil {
		return err
	}

	return renderTopics(doc, sentId, isTopicFile, opts, ui)
}

func topicsDocDB(docId int, sentId int, isTopicFile bool, opts TopicsOptions, ui UI) error {
	// For now, mirroring statDocDB behavior as requested
	// If we want to use the current DocHandler (filesystem-based "DB"), we could call it here,
	// but following the pattern in stat.go:
	return fmt.Errorf("database mode not implemented")
}

func renderTopics(doc sent.Doc, sentId int, isTopicFile bool, opts TopicsOptions, ui UI) error {
	if sentId < 0 || sentId >= len(doc.Tokens) {
		return fmt.Errorf("sentence index %d out of range (0-%d)", sentId, len(doc.Tokens)-1)
	}

	s := doc.Tokens[sentId]
	// Treat the single sentence as a document for matching
	matchDoc := sent.Doc{Tokens: [][]sent.Token{s}}

	r := render.NewRenderer()
	r.HasColor = false

	prefix := fmt.Sprintf("%54s", render.Yellow256+render.Off) + "✍  "
	r.Sentence(s, prefix)
	fmt.Fprintln(ui.Out)

	th, err := getTopicHandler(opts.TopicPath, isTopicFile)
	if err != nil {
		return err
	}

	allTopics, err := th.All()
	if err != nil {
		return err
	}

	r.HasColor = true
	r.HasPrefix = true
	r.PrefixDocFunc = render.PrefixFuncEmpty
	r.Format = opts.Format

	for _, tp := range allTopics {
		matcher := match.NewMatcher(tp)
		matcher.Match(matchDoc)
		res := matcher.Sentences()

		if len(res) == 0 {
			continue
		}

		r.Match(res)
	}

	return nil
}
