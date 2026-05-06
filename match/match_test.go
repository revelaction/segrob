package match

import (
	"testing"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

func TestMatchSentenceEmptyExpr(t *testing.T) {
	m := NewMatcher(topic.TopicExpr{})
	s := sent.Sentence{
		Tokens: []sent.Token{{Lemma: "casa", Index: 0}},
	}

	sm := m.MatchSentence(s)
	if sm != nil {
		t.Fatal("expected nil for empty expression")
	}
}

func TestMatchSentenceSingleLemma(t *testing.T) {
	expr := topic.TopicExpr{
		{Lemma: "casa"},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Lemma: "el", Index: 0},
			{Lemma: "casa", Index: 1},
			{Lemma: "grande", Index: 2},
		},
	}

	sm := m.MatchSentence(s)
	if sm == nil {
		t.Fatal("expected match")
	}

	if sm.Expr != "casa" {
		t.Fatalf("expected Expr 'casa', got '%s'", sm.Expr)
	}

	if len(sm.Tokens) != 1 {
		t.Fatalf("expected 1 occurrence, got %d", len(sm.Tokens))
	}

	if sm.Tokens[0][0].Lemma != "casa" {
		t.Fatalf("expected lemma casa, got %s", sm.Tokens[0][0].Lemma)
	}
}

func TestMatchSentenceNoMatch(t *testing.T) {
	expr := topic.TopicExpr{
		{Lemma: "perro"},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Lemma: "el", Index: 0},
			{Lemma: "casa", Index: 1},
		},
	}

	sm := m.MatchSentence(s)
	if sm != nil {
		t.Fatal("expected nil for non-matching sentence")
	}
}

func TestMatchSentenceNearChain(t *testing.T) {
	expr := topic.TopicExpr{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 3},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Lemma: "tomar", Index: 0},
			{Lemma: "la", Index: 1},
			{Lemma: "mano", Index: 2},
		},
	}

	sm := m.MatchSentence(s)
	if sm == nil {
		t.Fatal("expected match for near chain")
	}

	if len(sm.Tokens) != 1 {
		t.Fatalf("expected 1 occurrence, got %d", len(sm.Tokens))
	}

	if len(sm.Tokens[0]) != 2 {
		t.Fatalf("expected chain of 2 tokens, got %d", len(sm.Tokens[0]))
	}
}

func TestMatchSentenceNearTooFar(t *testing.T) {
	expr := topic.TopicExpr{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 2},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Lemma: "tomar", Index: 0},
			{Lemma: "la", Index: 1},
			{Lemma: "buena", Index: 2},
			{Lemma: "mano", Index: 3},
		},
	}

	sm := m.MatchSentence(s)
	if sm != nil {
		t.Fatal("expected nil, mano is beyond near=2")
	}
}

func TestMatchSentenceTagMatch(t *testing.T) {
	expr := topic.TopicExpr{
		{Tag: "VerbForm=Inf"},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Tag: "VerbForm=Inf|Mood=Sub", Index: 0, Lemma: "ser"},
		},
	}

	sm := m.MatchSentence(s)
	if sm == nil {
		t.Fatal("expected match for tag contains")
	}
}

func TestMatchSentenceOrLemma(t *testing.T) {
	expr := topic.TopicExpr{
		{Lemma: "casa|hogar"},
	}
	m := NewMatcher(expr)

	s := sent.Sentence{
		Tokens: []sent.Token{
			{Lemma: "hogar", Index: 0},
		},
	}

	sm := m.MatchSentence(s)
	if sm == nil {
		t.Fatal("expected match for OR lemma")
	}
}

func TestAllTokens(t *testing.T) {
	sm := &SentenceMatch{
		Tokens: [][]sent.Token{
			{{Lemma: "a", Index: 0}, {Lemma: "b", Index: 1}},
			{{Lemma: "c", Index: 2}, {Lemma: "d", Index: 3}},
		},
	}

	all := sm.AllTokens()
	if len(all) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(all))
	}
}
