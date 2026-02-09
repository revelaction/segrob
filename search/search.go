package search

import (
	"errors"

	"github.com/revelaction/segrob/match"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
)

// Search orchestrates the strategy selection for finding sentences
// that match a topic expression against a document repository.
type Search struct {
	topic  topic.Topic
	repo   storage.DocRepository
	docID  *int
	labels []string
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

// WithLabels restricts the search to documents matching ALL given labels.
func (s *Search) WithLabels(labels []string) *Search {
	s.labels = labels
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

		m := match.NewMatcher(s.topic)
		m.AddTopicExpr(expr)

		for _, sentence := range doc.Sentences {
			sm := m.MatchSentence(sentence)
			if sm != nil {
				if err := onMatch(sm); err != nil {
					return cursor, err
				}
			}
		}
		return cursor, nil
	}

	// Strategy 2: Find candidated (indexed search)
	lemmas := expr.Lemmas()
	if len(lemmas) == 0 {
		return cursor, errors.New("expression must contain at least one lemma for indexing")
	}

	m := match.NewMatcher(s.topic)
	m.AddTopicExpr(expr)

	return s.repo.FindCandidates(lemmas, s.labels, cursor, limit, func(sentence sent.Sentence) error {
		sm := m.MatchSentence(sentence)
		if sm != nil {
			// Expecting at most one match per sentence
			return onMatch(sm)
		}
		return nil
	})
}
