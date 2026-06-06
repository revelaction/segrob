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

const (
	// maxBatchSize limits the number of candidates fetched from storage in a
	// single FindCandidates call. It protects against SQLite expression limits
	// (massive IN clauses) and prevents excessive memory allocation spikes
	// during JSON unmarshaling.
	maxBatchSize = 500
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

//
// 1. Shuffle topic.Exprs
// 2. M = MinExpressions (or len(exprs) if not set)
// 3. [minRowid, maxRowid] = SentenceRowidRange(LabelID)
// 4. results = []
// 5. for i, expr in shuffled_exprs:
//      a. randomCursor = rand(minRowid, maxRowid)
//      b. fetch candidates (budget: CandidateBudget), match each:
//         - on match: append to results
//      c. if cursor exhausted and started > minRowid: wrap to minRowid, continue
//      d. if i >= M and len(results) >= Size: break
// 6. shuffle results, trim to Size
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
		lemmas := expr.Lemmas()
		if len(lemmas) > 0 {
			m := match.NewMatcher(expr)

			// Pick a random entry point so each expression starts from a
			// different position, avoiding positional bias across expressions.
			randomCursor := minRowid + rand.Int64N(maxRowid-minRowid+1)

			// Forward scan: from randomCursor to the end of the book.
			forward, forwardFetched, err := s.scanRange(
				lemmas, m,
				storage.Cursor(randomCursor-1), maxRowid,
				s.opts.CandidateBudget,
			)
			if err != nil {
				return nil, fmt.Errorf("sample/topic: expr %d forward: %w", i, err)
			}

			results = append(results, forward...)

			// Wrap scan: from the start of the book to randomCursor, spending
			// only the budget not consumed by the forward scan.
			remaining := s.opts.CandidateBudget - forwardFetched
			if remaining > 0 {
				wrap, _, err := s.scanRange(
					lemmas, m,
					storage.Cursor(minRowid-1), randomCursor,
					remaining,
				)
				if err != nil {
					return nil, fmt.Errorf("sample/topic: expr %d wrap: %w", i, err)
				}

				results = append(results, wrap...)
			}
		}

		if i+1 >= minExprs && len(results) >= s.opts.Size {
			break
		}
	}

	// Final partial shuffle to randomly select 'Size' matches.
	// We use a partial Fisher-Yates shuffle which only shuffles the first
	// 'k' positions (where k = size). Since each iteration selects a random
	// element from the remaining pool [i, n) and swaps it into position i,
	// stopping after 'k' iterations yields a uniformly random sample.
	// This reduces the time complexity from O(n) (full shuffle) to O(k),
	// which is a meaningful optimization when len(results) >> s.opts.Size.
	size := s.opts.Size
	if size > len(results) {
		size = len(results)
	}

	for i := 0; i < size; i++ {
		j := i + rand.IntN(len(results)-i)
		results[i], results[j] = results[j], results[i]
	}

	return results[:size], nil
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
	batchSize := maxBatchSize
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
