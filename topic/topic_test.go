package topic

import (
	"testing"
)

func TestParseSimpleLemma(t *testing.T) {
	expr, err := Parse([]string{"casa"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expr) != 1 {
		t.Fatalf("expected 1 item, got %d", len(expr))
	}

	if expr[0].Lemma != "casa" {
		t.Fatalf("expected lemma casa, got %s", expr[0].Lemma)
	}
}

func TestParseLemmaWithNear(t *testing.T) {
	expr, err := Parse([]string{"tomar", "3", "mano"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expr) != 2 {
		t.Fatalf("expected 2 items, got %d", len(expr))
	}

	if expr[1].Near != 3 {
		t.Fatalf("expected Near=3, got %d", expr[1].Near)
	}

	if expr[1].Lemma != "mano" {
		t.Fatalf("expected lemma mano, got %s", expr[1].Lemma)
	}
}

func TestParseTag(t *testing.T) {
	expr, err := Parse([]string{"NOUN"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if expr[0].Tag != "NOUN" {
		t.Fatalf("expected tag NOUN, got %s", expr[0].Tag)
	}

	if expr[0].Lemma != "" {
		t.Fatalf("expected empty lemma, got %s", expr[0].Lemma)
	}
}

func TestParseErrorFirstNumber(t *testing.T) {
	_, err := Parse([]string{"3", "casa"})
	if err == nil {
		t.Fatal("expected error for number as first argument")
	}
}

func TestParseErrorConsecutiveNumbers(t *testing.T) {
	_, err := Parse([]string{"casa", "3", "4"})
	if err == nil {
		t.Fatal("expected error for consecutive numbers")
	}
}

func TestLemmas(t *testing.T) {
	expr := TopicExpr{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 3},
		{Tag: "NOUN"},
		{Lemma: "tomar"},
	}

	lemmas := expr.Lemmas()
	if len(lemmas) != 2 {
		t.Fatalf("expected 2 unique lemmas, got %d", len(lemmas))
	}
}

func TestExprString(t *testing.T) {
	expr := TopicExpr{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 2},
	}

	s := expr.String()
	if s != "tomar 2 mano" {
		t.Fatalf("expected 'tomar 2 mano', got '%s'", s)
	}
}

func TestEqualExpr(t *testing.T) {
	a := TopicExpr{{Lemma: "casa"}, {Lemma: "grande", Near: 2}}
	b := TopicExpr{{Lemma: "casa"}, {Lemma: "grande", Near: 2}}

	if !EqualExpr(a, b) {
		t.Fatal("expected equal expressions")
	}
}

func TestEqualExprDifferent(t *testing.T) {
	a := TopicExpr{{Lemma: "casa"}}
	b := TopicExpr{{Lemma: "perro"}}

	if EqualExpr(a, b) {
		t.Fatal("expected different expressions")
	}
}
