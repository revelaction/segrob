package match

import (
	"strings"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

// Matcher matches a single expression against sentences.
type Matcher struct {
	Expr topic.TopicExpr
}

// SentenceMatch represents the result of matching one expression against one sentence.
type SentenceMatch struct {
	// Tokens contains the list of match occurrences. Each []sent.Token is one
	// complete match with tokens ordered by item position in the expression.
	Tokens [][]sent.Token `json:"tokens"`

	// Sentence is the matched sentence
	Sentence sent.Sentence `json:"sentence"`

	// Expr is the human-readable expression that produced this match
	// (e.g. "tener 2 proposito"). Set by the matcher via TopicExpr.String().
	Expr string `json:"expr"`

	// TopicName is the topic this expression belongs to.
	// Not set by the matcher. Callers set this when topic context is available.
	TopicName string `json:"topic_name,omitempty"`
}

// AllTokens returns all matched tokens from all match occurrences, flattened.
func (sm *SentenceMatch) AllTokens() []sent.Token {
	var all []sent.Token
	for _, tks := range sm.Tokens {
		all = append(all, tks...)
	}
	return all
}

// MatchSentence matches the expression against a single sentence.
// Returns nil if the expression does not match.
func (m *Matcher) MatchSentence(sentence sent.Sentence) *SentenceMatch {
	if len(m.Expr) == 0 {
		return nil
	}

	tokens := matchExpr(sentence.Tokens, m.Expr)
	if tokens == nil {
		return nil
	}

	return &SentenceMatch{
		Tokens:   tokens,
		Sentence: sentence,
		Expr:     m.Expr.String(),
	}
}

// matchExpr matches a single TopicExpr against a sentence.
// Returns the list of match occurrences (each is a chain of tokens in item order).
// Returns nil if the expression does not match.
func matchExpr(sentence []sent.Token, expr topic.TopicExpr) [][]sent.Token {
	// candidates tracks the current set of partial match chains.
	// After processing item i, each chain contains i+1 tokens.
	var candidates [][]sent.Token

	for i, item := range expr {
		if i == 0 {
			// First item: independent match, each hit starts a new chain
			for _, t := range sentence {
				if isTokenMatch(t, item) {
					candidates = append(candidates, []sent.Token{t})
				}
			}
			if len(candidates) == 0 {
				return nil
			}
			continue
		}

		// Items 1..n: must have Near > 0, extend existing candidates
		var extended [][]sent.Token
		sentenceEnd := len(sentence) - 1

		for _, chain := range candidates {
			lastToken := chain[len(chain)-1]
			if lastToken.Index >= sentenceEnd {
				continue
			}

			end := lastToken.Index + item.Near
			if end > sentenceEnd {
				end = sentenceEnd
			}

			for _, t := range sentence[lastToken.Index+1 : end+1] {
				if isTokenMatch(t, item) {
					newChain := make([]sent.Token, len(chain), len(chain)+1)
					copy(newChain, chain)
					newChain = append(newChain, t)
					extended = append(extended, newChain)
				}
			}
		}

		if len(extended) == 0 {
			return nil
		}
		candidates = extended
	}

	return candidates
}

func NewMatcher(expr topic.TopicExpr) *Matcher {
	return &Matcher{
		Expr: expr,
	}
}

func isTokenMatch(t sent.Token, item topic.TopicExprItem) bool {
	//
	// Lemma field
	//
	if len(item.Lemma) > 0 {
		// optimistically try to split possible OR values
		// If no "|" just one value
		isOrValue := false
		for _, orValue := range strings.Split(item.Lemma, "|") {
			if orValue == t.Lemma {
				isOrValue = true
				break
			}
		}

		if !isOrValue {
			return false
		}
	}

	//
	// Tag field
	//
	if len(item.Tag) > 0 {
		switch separator(item.Tag) {
		case "|":
			// OR
			isMatched := false
			for _, orItem := range strings.Split(item.Tag, "|") {
				if strings.Contains(t.Tag, orItem) {
					isMatched = true
					break
				}
			}

			if !isMatched {
				return false
			}
		case "+":
			// AND: must contain each
			for _, andItem := range strings.Split(item.Tag, "+") {
				if !strings.Contains(t.Tag, andItem) {
					return false
				}
			}
		default:
			// single
			if !strings.Contains(t.Tag, item.Tag) {
				return false
			}
		}
	}

	if len(item.Pos) > 0 {
		if item.Pos != t.Pos {
			return false
		}
	}

	return true
}

func separator(field string) string {
	if strings.Contains(field, "|") {
		return "|"
	}

	if strings.Contains(field, "+") {
		return "+"
	}

	return ""
}
