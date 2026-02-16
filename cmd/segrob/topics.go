package main

import (
	"fmt"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func topicsCommand(docRepo storage.DocRepository, topicRepo storage.TopicRepository, opts TopicsOptions, docId int, sentId int, ui UI) error {
	if p, ok := docRepo.(storage.Preloader); ok {
		if err := p.LoadNLP(nil, &docId, nil); err != nil {
			return err
		}
	}
	doc, err := docRepo.Read(docId)
	if err != nil {
		return err
	}

	return renderTopics(doc, sentId, topicRepo, opts, ui)
}

func renderTopics(doc sent.Doc, sentId int, topicRepo storage.TopicRepository, opts TopicsOptions, ui UI) error {
	if sentId < 0 || sentId >= len(doc.Sentences) {
		return fmt.Errorf("sentence index %d out of range (0-%d)", sentId, len(doc.Sentences)-1)
	}

	s := doc.Sentences[sentId]

	r := render.NewRenderer()
	r.HasColor = false

	prefix := fmt.Sprintf("%54s", render.Yellow256+render.Off) + "✍  "
	r.Sentence(s.Tokens, prefix)
	fmt.Fprintln(ui.Out)

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

		r.Match(res)
	}

	return nil
}