package match

import (
	"strings"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

// Matcher matchs a Doc (or a set of Docs) against a Topic (+ ArgExpr)
type Matcher struct {
	Topic topic.Topic

	// ArgExpr is an additional topic expresion passed as argument to the
	// command line
	// ArgExpr have an AND semantic, they must match the sentence in addition
	// to one or more TopicExpr of the Topic.
	ArgExpr topic.TopicExpr
}

// ExprMatch is the result of matching one TopicExpr against one Sentence.
// Tokens contains the list of match occurrences. Each []sent.Token is one
// complete match with tokens ordered by item position in the expression.
type ExprMatch struct {
	ExprIndex int            // position of the expression in Topic.Exprs
	Tokens    [][]sent.Token // each element is one full match occurrence
}

// SentenceMatch represents a sentence matched by one or more TopicExpr.
type SentenceMatch struct {
	// topicName is the topic that matched this sentence
	topicName string

	// matches contains one ExprMatch per matched expression
	matches []ExprMatch

	// Sentence is the matched sentence
	Sentence sent.Sentence

	// NumExprs is the number of topicExpr that matched. Used to sort results.
	NumExprs int
}

// AllTokens returns all matched tokens from all expressions, flattened.
func (sm *SentenceMatch) AllTokens() []sent.Token {
	var all []sent.Token
	for _, em := range sm.matches {
		for _, tks := range em.Tokens {
			all = append(all, tks...)
		}
	}
	return all
}

// TopicName return the topic name of the sentence match
// used by the render to prefix the topic in the `topics` command.
func (sm *SentenceMatch) TopicName() string {
	return sm.topicName
}

// MatchSentence matches a posible Topic AND a possible TopicExpr for a given sentence.
func (m *Matcher) MatchSentence(sentence sent.Sentence) *SentenceMatch {
	hasTopic := len(m.Topic.Exprs) > 0
	hasExpr := len(m.ArgExpr) > 0

	sm := &SentenceMatch{}

	// ArgExpr check (AND gate)
	if hasExpr {
		tokens := matchExpr(sentence.Tokens, m.ArgExpr)
		if tokens == nil {
			return nil
		}
		sm.matches = append(sm.matches, ExprMatch{ExprIndex: -1, Tokens: tokens})
	}

	// Topic expressions (OR)
	for i, expr := range m.Topic.Exprs {
		tokens := matchExpr(sentence.Tokens, expr)
		if tokens != nil {
			sm.NumExprs++
			sm.matches = append(sm.matches, ExprMatch{ExprIndex: i, Tokens: tokens})
		}
	}

	if hasTopic && sm.NumExprs == 0 {
		return nil
	}

	sm.topicName = m.Topic.Name
	sm.Sentence = sentence

	return sm
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

func (m *Matcher) AddTopicExpr(expr topic.TopicExpr) {
	m.ArgExpr = expr
}

func NewMatcher(topic topic.Topic) *Matcher {
	return &Matcher{
		Topic: topic,
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
