// sample/topic/topic.go
package topic

import (
	"fmt"
	"math/rand/v2"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/sample"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	t "github.com/revelaction/segrob/topic"
)

// Options configures the topic sampling algorithm.
type Options struct {
	// Size is the target number of total matches to return.
	Size int

	// MinExpressions is the minimum number of topic expressions to query
	// before the algorithm is allowed to stop early. This guarantees that
	// rare expressions get a chance even when common ones produce many matches.
	MinExpressions int

	// CandidateBudget limits the total number of candidates fetched from
	// storage per expression. Bounding this prevents rare expressions
	// from causing full-book scans.
	CandidateBudget int

	// LabelID scopes the search to a specific book (label).
	LabelID int
}

// New creates a sample.Sampler that dynamically extracts sentences
// matching the given topic within a single document.
func New(dr storage.DocReader, tp t.Topic, opts Options) sample.Sampler {
	return &sampler{
		dr:   dr,
		tp:   tp,
		opts: opts,
	}
}

type sampler struct {
	dr   storage.DocReader
	tp   t.Topic
	opts Options
}

func (s *sampler) Sample() ([]*match.SentenceMatch, error) {
	if len(s.tp.Exprs) == 0 {
		return nil, nil
	}

	minRowid, maxRowid, err := s.dr.SentenceRowidRange(s.opts.LabelID)
	if err != nil {
		return nil, fmt.Errorf("sample/topic: rowid range: %w", err)
	}

	if minRowid == 0 && maxRowid == 0 {
		return nil, nil
	}

	// Shuffle expressions to randomize which are tried first.
	exprs := make([]t.TopicExpr, len(s.tp.Exprs))
	copy(exprs, s.tp.Exprs)
	rand.Shuffle(len(exprs), func(i, j int) {
		exprs[i], exprs[j] = exprs[j], exprs[i]
	})

	minExprs := s.opts.MinExpressions
	if minExprs <= 0 || minExprs > len(exprs) {
		minExprs = len(exprs)
	}

	var results []*match.SentenceMatch

	for i, expr := range exprs {
		exprMatches, err := s.scanExpression(expr, minRowid, maxRowid)
		if err != nil {
			return nil, fmt.Errorf("sample/topic: expr %d: %w", i, err)
		}

		results = append(results, exprMatches...)

		if i+1 >= minExprs && len(results) >= s.opts.Size {
			break
		}
	}

	// Final shuffle to mix matches from different expressions.
	rand.Shuffle(len(results), func(i, j int) {
		results[i], results[j] = results[j], results[i]
	})

	if len(results) > s.opts.Size {
		results = results[:s.opts.Size]
	}

	return results, nil
}

// scanExpression scans the book for candidates matching the given expression,
// starting from a random cursor position within [minRowid, maxRowid].
// It performs a forward scan and, if the budget is not exhausted, wraps
// around to cover the beginning of the book.
func (s *sampler) scanExpression(
	expr t.TopicExpr,
	minRowid int64,
	maxRowid int64,
) ([]*match.SentenceMatch, error) {

	lemmas := expr.Lemmas()
	if len(lemmas) == 0 {
		return nil, nil
	}

	m := match.NewMatcher(expr)
	budget := s.opts.CandidateBudget

	randomCursor := minRowid + rand.Int64N(maxRowid-minRowid+1)

	// Phase 1: Forward scan from randomCursor toward maxRowid.
	forward, forwardFetched, err := s.scanRange(
		lemmas, m,
		storage.Cursor(randomCursor-1), maxRowid,
		budget,
	)
	if err != nil {
		return forward, err
	}

	// Phase 2: Wrap-around from minRowid to randomCursor.
	// Only if the forward scan did not exhaust the budget.
	remaining := budget - forwardFetched
	if remaining > 0 {
		wrap, _, err := s.scanRange(
			lemmas, m,
			storage.Cursor(minRowid-1), randomCursor,
			remaining,
		)
		if err != nil {
			return append(forward, wrap...), err
		}
		forward = append(forward, wrap...)
	}

	return forward, nil
}

// scanRange fetches candidates for the given lemmas, starting from cursor,
// and matches them against the Matcher. It stops when the candidate budget
// is exhausted or when a candidate's Rowid exceeds maxRowid.
//
// Returns the collected matches and the number of candidates examined.
func (s *sampler) scanRange(
	lemmas []string,
	m *match.Matcher,
	cursor storage.Cursor,
	maxRowid int64,
	budget int,
) ([]*match.SentenceMatch, int, error) {

	var matches []*match.SentenceMatch
	batchSize := 500
	if budget < batchSize {
		batchSize = budget
	}

	labelIDs := []int{s.opts.LabelID}
	fetched := 0

	for budget > 0 {
		batchFetched := 0
		newCursor, err := s.dr.FindCandidates(lemmas, labelIDs, cursor, batchSize, func(ss sent.Sentence) error {
			if ss.Rowid > maxRowid {
				return storage.ErrStopScan
			}

			batchFetched++

			sm := m.MatchSentence(ss)
			if sm != nil {
				sm.TopicName = s.tp.Name
				matches = append(matches, sm)
			}
			return nil
		})
		if err != nil {
			return matches, fetched, err
		}

		fetched += batchFetched
		budget -= batchFetched

		// No progress — the scan is exhausted for this lemma set.
		if newCursor == cursor {
			break
		}

		cursor = newCursor

		if batchSize > budget {
			batchSize = budget
		}
	}

	return matches, fetched, nil
}
