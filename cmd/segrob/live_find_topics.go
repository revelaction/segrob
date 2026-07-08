package main

import (
	"fmt"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
)

func liveFindTopicsCommand(docRepo storage.DocRepository, topicRepo storage.TopicRepository, opts LiveFindTopicsOptions, docId string, sentId int, ui UI) error {
	zero := 0
	sentences, err := docRepo.Nlp(docId, sentId, &zero)
	if err != nil {
		return err
	}

	if len(sentences) == 0 {
		return fmt.Errorf("sentence index %d not found", sentId)
	}

	return renderTopics(sentences[0], topicRepo, opts, ui)
}

func renderTopics(s sent.Sentence, topicRepo storage.TopicRepository, opts LiveFindTopicsOptions, ui UI) error {
	r := render.NewCLIRenderer()
	r.HasColor = false

	prefix := fmt.Sprintf("%54s", render.Yellow256+render.Off) + "✍  "
	r.Sentence(s.Tokens, prefix)
	_, err := fmt.Fprintln(ui.Out)
	if err != nil {
		return err
	}

	allTopics, err := topicRepo.ReadAll("")
	if err != nil {
		return err
	}

	r.HasColor = true
	r.HasPrefix = true
	r.PrefixDocFunc = render.PrefixFuncEmpty
	r.Format = opts.Format

	for _, tp := range allTopics {
		for _, expr := range tp.Exprs {
			matcher := match.NewMatcher(expr)
			sm := matcher.MatchSentence(s)
			if sm == nil {
				continue
			}

			sm.TopicName = tp.Name
			res := []*match.SentenceMatch{sm}
			r.Render(res)
		}
	}

	return nil
}
