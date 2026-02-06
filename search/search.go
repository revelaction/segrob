package search

import (
	"errors"
	"fmt"

	"github.com/revelaction/segrob/match"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
)

// Search orchestrates the strategy selection for finding sentences
// that match a topic expression against a document repository.
type Search struct {
	topic topic.Topic
	repo  storage.DocRepository
	docID *int
}

// New creates a new Search instance with the given topic and repository.
// The topic is used to construct the internal Matcher for evaluating expressions.
func New(t topic.Topic, dr storage.DocRepository) *Search {
	return &Search{
		topic: t,
		repo:  dr,
	}
}

// WithDocID restricts the search to a single document ID.
// If set, the single-document strategy (Read) will be favored over
// the indexed strategy (FindCandidates).
func (s *Search) WithDocID(id int) *Search {
	s.docID = &id
	return s
}

// Sentences returns matched sentences for the given expression, handling pagination.
func (s *Search) Sentences(expr topic.TopicExpr, cursor storage.Cursor, limit int, onMatch func(*match.SentenceMatch) error) (storage.Cursor, error) {
	// Strategy 1: Single Document (No Index)
	if s.docID != nil {
		doc, err := s.repo.Read(*s.docID)
		if err != nil {
			return cursor, err
		}
		// Ensure doc has ID set (Read might return 0 if backend doesn't populate)
		doc.Id = *s.docID

		matcher := match.NewMatcher(s.topic)
		matcher.AddTopicExpr(expr)
		matcher.Match(doc)

		for _, m := range matcher.Sentences() {
			if err := onMatch(m); err != nil {
				return cursor, err
			}
		}
		return cursor, nil
	}

	// Strategy 2: Find candidated (indexed search)
	lemmas := expr.Lemmas()
	if len(lemmas) == 0 {
		return cursor, errors.New("expression must contain at least one lemma for indexing")
	}

	// Fetch titles for result enhancement
	docMap := make(map[int]string)
	docs, err := s.repo.List()
	if err != nil {
		return cursor, fmt.Errorf("failed to list docs: %w", err)
	}
	for _, d := range docs {
		docMap[d.Id] = d.Title
	}

	return s.repo.FindCandidates(lemmas, cursor, limit, func(res storage.SentenceResult) error {
		doc := sent.Doc{
			Id:     res.DocID,
			Title:  docMap[res.DocID],
			Tokens: [][]sent.Token{res.Tokens},
		}

		// Use a fresh matcher for each sentence to be stateless and avoid accumulation
		// TODO indefficient, make new match.MatchSentence with zero llocations,
		// needs also sentenceExprMatch zero allocations for map
		m := match.NewMatcher(s.topic)
		m.AddTopicExpr(expr)
		m.Match(doc)

		matches := m.Sentences()
		if len(matches) > 0 {
			// Expecting at most one match per sentence
			return onMatch(matches[0])
		}
		return nil
	})
}
