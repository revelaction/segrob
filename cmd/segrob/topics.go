package main

import (
	"fmt"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func liveFindTopicsCommand(docRepo storage.DocRepository, topicRepo storage.TopicRepository, opts LiveFindTopicsOptions, docId string, sentId int, ui UI) error {
	sentences, err := docRepo.Nlp(docId)
	if err != nil {
		return err
	}

	return renderTopics(sentences, sentId, topicRepo, opts, ui)
}

func renderTopics(sentences []sent.Sentence, sentId int, topicRepo storage.TopicRepository, opts LiveFindTopicsOptions, ui UI) error {
	if sentId < 0 || sentId >= len(sentences) {
		return fmt.Errorf("sentence index %d out of range (0-%d)", sentId, len(sentences)-1)
	}

	s := sentences[sentId]

	r := render.NewCLIRenderer()
	r.HasColor = false

	prefix := fmt.Sprintf("%54s", render.Yellow256+render.Off) + "✍  "
	r.Sentence(s.Tokens, prefix)
	_, err := fmt.Fprintln(ui.Out)
	if err != nil {
		return err
	}

	allTopics, err := topicRepo.ReadAll()
	if err != nil {
		return err
	}

	r.HasColor = true
	r.HasPrefix = true
	r.PrefixDocFunc = render.PrefixFuncEmpty
	r.Format = opts.Format

	for _, tp := range allTopics {
		matcher := match.NewMatcher(tp)
		sm := matcher.MatchSentence(s)
		if sm == nil {
			continue
		}

		res := []*match.SentenceMatch{sm}

		r.Render(res)
	}

	return nil
}
