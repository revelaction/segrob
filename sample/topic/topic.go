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

	// BalancedRatio controls the trade-off between expression diversity
	// and natural corpus distribution in the final selection.
	//
	// It specifies what fraction of Size is allocated equally across all
	// matched expressions (the "balanced bucket"), while the remaining
	// fraction is filled by random selection proportional to each
	// expression's natural frequency (the "distribution bucket").
	//
	// Example: Size=50, BalancedRatio=0.6, 10 distinct expressions:
	//   Balanced bucket:     30 slots → 3 per expression (diversity)
	//   Distribution bucket: 20 slots → random, favoring common expressions
	//
	// A value of 0 disables balanced selection (pure distribution).
	BalancedRatio float64
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

	return selectBalanced(results, s.opts.Size, s.opts.BalancedRatio), nil
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
	// we shrink the batch size immediately so we don't waste memory and
	// database resources fetching records we won't be allowed to process.
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

        // ensures that subsequent queries respect the remaining budget. For
        // example, if you start with a budget of 600 and a max batch size of
        // 500, the first loop iteration fetches 500 records. You now have a
        // remaining budget of 100. If we don't reduce the batchSize at the
        // end of the loop, the next iteration will ask the database for
        // another 500 records, over-fetching by 400. Adjusting it here
        // guarantees the final query perfectly matches the exact remaining
        // budget.
		if batchSize > budget {
			batchSize = budget
		}
	}

	return matches, fetched, nil
}

// selectBalanced picks `size` matches from collected results using a
// two-bucket strategy that balances expression diversity against natural
// corpus distribution.
//
//	Bucket 1 — Balanced (ratio × size slots):
//	  Slots are divided equally among all distinct expressions. Every
//	  expression that produced matches receives at least one slot (budget
//	  permitting). If an expression has fewer matches than its quota,
//	  the shortfall is redistributed round-robin to expressions with
//	  surplus. This prevents common expressions from drowning out rare ones.
//
//	Bucket 2 — Distribution (remaining slots):
//	  Filled by uniform random selection from matches not consumed by
//	  bucket 1. Because common expressions naturally produce more matches,
//	  they dominate this portion — preserving the corpus's frequency
//	  signal for the learner.
//
// Example: size=50, ratio=0.6, 10 distinct expressions
//
//	Balanced:     30 slots → 3 per expression (guaranteed diversity)
//	Distribution: 20 slots → random, favoring frequent expressions
//
// If ratio is 0, falls back to a partial Fisher-Yates (pure distribution).
func selectBalanced(results []*match.SentenceMatch, size int, ratio float64) []*match.SentenceMatch {
	if size > len(results) {
		size = len(results)
	}

	if size == 0 {
		return nil
	}

	// Fallback: ratio=0 disables balanced selection. Use a simple partial
	// Fisher-Yates for backward compatibility (pure distribution).
	if ratio <= 0 {
		for i := 0; i < size; i++ {
			j := i + rand.IntN(len(results)-i)
			results[i], results[j] = results[j], results[i]
		}
		return results[:size]
	}

	// --- Group matches by expression ---

	type exprGroup struct {
		matches []*match.SentenceMatch
		taken   int // how many matches have been selected
	}

	groupIndex := make(map[string]int)
	var groups []*exprGroup

	for _, m := range results {
		i, ok := groupIndex[m.Expr]
		if !ok {
			i = len(groups)
			groupIndex[m.Expr] = i
			groups = append(groups, &exprGroup{})
		}
		groups[i].matches = append(groups[i].matches, m)
	}

	// Shuffle each group internally so picks within an expression are random.
	for _, g := range groups {
		rand.Shuffle(len(g.matches), func(i, j int) {
			g.matches[i], g.matches[j] = g.matches[j], g.matches[i]
		})
	}

	// Shuffle group order for fairness when distributing remainder quotas.
	rand.Shuffle(len(groups), func(i, j int) {
		groups[i], groups[j] = groups[j], groups[i]
	})

	n := len(groups)

	// --- Bucket 1: Balanced allocation ---
	//
	// Divide the balanced budget equally, distributing the integer
	// remainder to the first groups (which are already shuffled).
	//
	// Example: balancedBudget=12, n=5 → perExpr=2, remainder=2
	//   Groups 0,1 get quota=3; groups 2,3,4 get quota=2. Total=12.
	balancedBudget := int(float64(size) * ratio)
	perExpr := balancedBudget / n
	remainder := balancedBudget % n

	var selected []*match.SentenceMatch
	shortfall := 0

	for i, g := range groups {
		quota := perExpr
		if i < remainder {
			quota++
		}

		take := quota
		if take > len(g.matches) {
			shortfall += take - len(g.matches)
			take = len(g.matches)
		}

		selected = append(selected, g.matches[:take]...)
		g.taken = take
	}

	// Redistribute shortfall: when some expressions couldn't fill their
	// quota (fewer matches than allocated), distribute the leftover slots
	// to expressions that still have unused matches. Round-robin ensures
	// no single expression absorbs all the surplus.
	for shortfall > 0 {
		advanced := false
		for _, g := range groups {
			if shortfall <= 0 {
				break
			}
			if g.taken < len(g.matches) {
				selected = append(selected, g.matches[g.taken])
				g.taken++
				shortfall--
				advanced = true
			}
		}
		if !advanced {
			break
		}
	}

	// --- Bucket 2: Distribution (natural frequency) ---
	//
	// Fill the remaining slots from matches not consumed by bucket 1.
	// Common expressions naturally have more leftover matches, so they
	// dominate this portion — preserving the corpus frequency signal.
	distributionBudget := size - len(selected)
	if distributionBudget > 0 {
		var pool []*match.SentenceMatch
		for _, g := range groups {
			if g.taken < len(g.matches) {
				pool = append(pool, g.matches[g.taken:]...)
			}
		}

		if distributionBudget > len(pool) {
			distributionBudget = len(pool)
		}

		// Partial Fisher-Yates on the remaining pool.
		for i := 0; i < distributionBudget; i++ {
			j := i + rand.IntN(len(pool)-i)
			pool[i], pool[j] = pool[j], pool[i]
		}

		selected = append(selected, pool[:distributionBudget]...)
	}

	// Final shuffle to interleave balanced and distribution entries,
	// removing any positional pattern from the two-phase selection.
	rand.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})

	return selected
}
