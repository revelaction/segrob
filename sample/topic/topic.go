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

    // MinExpressions is the minimum number of topic expressions that must
    // yield matches before the algorithm is allowed to stop early. This
    // guarantees that rare expressions get a chance even when common ones
    // produce many matches.
    MinExpressions int

    // MinSizePerExpression guarantees that up to this many matches are
    // reserved for every distinct expression found, before the remaining
    // slots are filled randomly from the pool of all leftover matches.
    MinSizePerExpression int

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

    // Fail-fast validation for the orthogonal constraints.
    if s.opts.MinSizePerExpression < 1 {
        return nil, fmt.Errorf("sample/topic: MinSizePerExpression must be at least 1")
    }
    if s.opts.MinExpressions*s.opts.MinSizePerExpression > s.opts.Size {
        return nil, fmt.Errorf("sample/topic: MinExpressions (%d) * MinSizePerExpression (%d) exceeds Size (%d)",
            s.opts.MinExpressions, s.opts.MinSizePerExpression, s.opts.Size)
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

    // Group results by expression right from the start.
    results := make(map[string][]*match.SentenceMatch)
    matchedExprs := 0
    totalMatches := 0

    for _, expr := range exprs {
        lemmas := expr.Lemmas()
        if len(lemmas) > 0 {
            m := match.NewMatcher(expr)
            key := expr.String()

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
                return nil, fmt.Errorf("sample/topic: expr %v forward: %w", expr, err)
            }

            if len(forward) > 0 {
                results[key] = append(results[key], forward...)
                totalMatches += len(forward)
            }
            hasMatch := len(forward) > 0

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
                    return nil, fmt.Errorf("sample/topic: expr %v wrap: %w", expr, err)
                }

                if len(wrap) > 0 {
                    results[key] = append(results[key], wrap...)
                    totalMatches += len(wrap)
                    hasMatch = true
                }
            }

            if hasMatch {
                matchedExprs++
            }
        }

        if matchedExprs >= minExprs && totalMatches >= s.opts.Size {
            break
        }
    }

    return selectDistributed(results, s.opts.Size, s.opts.MinSizePerExpression), nil
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

        // Ensures that subsequent queries respect the remaining budget.
        if batchSize > budget {
            batchSize = budget
        }
    }

    return matches, fetched, nil
}

// selectDistributed picks `size` matches from collected results using a
// two-phase construction algorithm that guarantees expression diversity.
//
//  1. Iterate through the map of matches grouped by expression.
//  2. For each expression, shuffle its matches. Take up to `minPerExpr`
//     matches and append them directly to the final result. Append any
//     leftover matches for that expression to a flat `pool` slice.
//  3. Calculate how many slots remain to reach `size`.
//  4. Use a partial Fisher-Yates shuffle on the `pool` to randomly pick
//     the exact number of remaining slots needed. Because common expressions
//     contribute more leftovers to the pool, they naturally win more of
//     these slots, preserving the corpus frequency signal.
//  5. Append the pool picks to the final result, and do a final shuffle
//     to interleave protected and unprotected matches.
func selectDistributed(results map[string][]*match.SentenceMatch, size int, minPerExpr int) []*match.SentenceMatch {
    totalMatches := 0
    for _, matches := range results {
        totalMatches += len(matches)
    }

    if totalMatches == 0 {
        return nil
    }

    if size > totalMatches {
        size = totalMatches
    }

    var selected []*match.SentenceMatch
    var pool []*match.SentenceMatch

    // 1 & 2. Allocate protected slots and build the unprotected pool.
    for _, matches := range results {
        // Shuffle within the expression to randomize protected picks.
        rand.Shuffle(len(matches), func(i, j int) {
            matches[i], matches[j] = matches[j], matches[i]
        })

        take := minPerExpr
        if take > len(matches) {
            take = len(matches)
        }

        // Append protected matches to the final result.
        selected = append(selected, matches[:take]...)

        // Append the rest to the unprotected pool.
        if take < len(matches) {
            pool = append(pool, matches[take:]...)
        }
    }

    // 3. Calculate how many more we need from the pool.
    needed := size - len(selected)
    if needed > len(pool) {
        needed = len(pool)
    }

    // 4. Partial Fisher-Yates on the pool to pick the needed amount.
    for i := 0; i < needed; i++ {
        j := i + rand.IntN(len(pool)-i)
        pool[i], pool[j] = pool[j], pool[i]
    }
    selected = append(selected, pool[:needed]...)

    // 5. Final shuffle to interleave protected and unprotected.
    rand.Shuffle(len(selected), func(i, j int) {
        selected[i], selected[j] = selected[j], selected[i]
    })

    return selected
}
