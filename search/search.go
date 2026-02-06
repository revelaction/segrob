package search

import (
	"errors"

	"github.com/revelaction/segrob/match"
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

		m := match.NewMatcher(s.topic)
		m.AddTopicExpr(expr)

		for i, sentence := range doc.Tokens {
			sm := m.MatchSentence(sentence, doc.Id)
			if sm != nil {
				// Use the loop index as the authoritative SentenceId for Strategy 1
				// to avoid identity collisions (the "Tártaro" bug).
				sm.SentenceId = i
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

	return s.repo.FindCandidates(lemmas, cursor, limit, func(res storage.SentenceResult) error {

		// Use a fresh matcher for each sentence to be stateless and avoid accumulation
		// TODO indefficient, make new match.MatchSentence with zero llocations,
		// needs also sentenceExprMatch zero allocations for map
		match := m.MatchSentence(res.Tokens, res.DocID)

		if match != nil {
			// Expecting at most one match per sentence
			return onMatch(match)
		}
		return nil
	})
}
