package topic

import (
	"testing"
)

func TestParseSimpleLemma(t *testing.T) {
	expr, err := Parse([]string{"casa"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expr.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(expr.Items))
	}

	if expr.Items[0].Lemma != "casa" {
		t.Fatalf("expected lemma casa, got %s", expr.Items[0].Lemma)
	}
}

func TestParseLemmaWithNear(t *testing.T) {
	expr, err := Parse([]string{"tomar", "3", "mano"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(expr.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(expr.Items))
	}

	if expr.Items[1].Near != 3 {
		t.Fatalf("expected Near=3, got %d", expr.Items[1].Near)
	}

	if expr.Items[1].Lemma != "mano" {
		t.Fatalf("expected lemma mano, got %s", expr.Items[1].Lemma)
	}
}

func TestParseTag(t *testing.T) {
	expr, err := Parse([]string{"NOUN"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if expr.Items[0].Tag != "NOUN" {
		t.Fatalf("expected tag NOUN, got %s", expr.Items[0].Tag)
	}

	if expr.Items[0].Lemma != "" {
		t.Fatalf("expected empty lemma, got %s", expr.Items[0].Lemma)
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
	expr := TopicExpr{Items: []TopicExprItem{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 3},
		{Tag: "NOUN"},
		{Lemma: "tomar"},
	}}

	lemmas := expr.Lemmas()
	if len(lemmas) != 2 {
		t.Fatalf("expected 2 unique lemmas, got %d", len(lemmas))
	}
}

func TestExprString(t *testing.T) {
	expr := TopicExpr{Items: []TopicExprItem{
		{Lemma: "tomar"},
		{Lemma: "mano", Near: 2},
	}}

	s := expr.String()
	if s != "tomar 2 mano" {
		t.Fatalf("expected 'tomar 2 mano', got '%s'", s)
	}
}

func TestEqualExpr(t *testing.T) {
	a := TopicExpr{Items: []TopicExprItem{{Lemma: "casa"}, {Lemma: "grande", Near: 2}}}
	b := TopicExpr{Items: []TopicExprItem{{Lemma: "casa"}, {Lemma: "grande", Near: 2}}}

	if !EqualExpr(a, b) {
		t.Fatal("expected equal expressions")
	}
}

func TestEqualExprDifferent(t *testing.T) {
	a := TopicExpr{Items: []TopicExprItem{{Lemma: "casa"}}}
	b := TopicExpr{Items: []TopicExprItem{{Lemma: "perro"}}}

	if EqualExpr(a, b) {
		t.Fatal("expected different expressions")
	}
}

func TestDeduplicateEmpty(t *testing.T) {
	result := Deduplicate(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty, got %d", len(result))
	}
}

func TestDeduplicateNoDuplicates(t *testing.T) {
	exprs := []TopicExpr{
		{Items: []TopicExprItem{{Lemma: "casa"}}},
		{Items: []TopicExprItem{{Lemma: "coche"}}},
	}
	result := Deduplicate(exprs)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}

func TestDeduplicateRemovesDuplicates(t *testing.T) {
	exprs := []TopicExpr{
		{Items: []TopicExprItem{{Lemma: "casa"}}},
		{Items: []TopicExprItem{{Lemma: "coche"}}},
		{Items: []TopicExprItem{{Lemma: "casa"}}}, // duplicate
	}
	result := Deduplicate(exprs)
	if len(result) != 2 {
		t.Fatalf("expected 2 after dedup, got %d", len(result))
	}
	if result[0].Items[0].Lemma != "casa" {
		t.Fatalf("first should be casa, got %s", result[0].Items[0].Lemma)
	}
	if result[1].Items[0].Lemma != "coche" {
		t.Fatalf("second should be coche, got %s", result[1].Items[0].Lemma)
	}
}

func TestDeduplicatePreservesFirstSeen(t *testing.T) {
	// Flagged should be ignored — the first occurrence wins
	exprs := []TopicExpr{
		{Items: []TopicExprItem{{Lemma: "casa"}}, Flagged: false},
		{Items: []TopicExprItem{{Lemma: "casa"}}, Flagged: true},
	}
	result := Deduplicate(exprs)
	if len(result) != 1 {
		t.Fatalf("expected 1 after dedup, got %d", len(result))
	}
	if result[0].Flagged {
		t.Fatal("expected first-seen Flagged=false to be preserved")
	}
}

func TestDeduplicateMultiItemExpr(t *testing.T) {
	exprs := []TopicExpr{
		{Items: []TopicExprItem{{Lemma: "tomar"}, {Lemma: "mano", Near: 3}}},
		{Items: []TopicExprItem{{Lemma: "tomar"}, {Lemma: "mano", Near: 3}}}, // duplicate
		{Items: []TopicExprItem{{Lemma: "tomar"}, {Lemma: "mano", Near: 1}}}, // different Near
	}
	result := Deduplicate(exprs)
	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
}
