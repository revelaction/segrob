package render

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/revelaction/segrob/match"
	sent "github.com/revelaction/segrob/sentence"
)

func TestJSONRendererRenderEmpty(t *testing.T) {
	var buf bytes.Buffer
	r := NewJSONRenderer(&buf)
	r.Render(nil)

	var results []*match.SentenceMatch
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestJSONRendererRenderOneResult(t *testing.T) {
	sm := &match.SentenceMatch{
		TopicName: "test-topic",
		NumExprs:  1,
		Matches: []match.ExprMatch{
			{
				ExprIndex: 0,
				Tokens: [][]sent.Token{
					{{Index: 0, Lemma: "cat", Text: "cat"}},
				},
			},
		},
		Sentence: sent.Sentence{
			Id:    5,
			DocId: 1,
			Tokens: []sent.Token{
				{Index: 0, Lemma: "cat", Text: "cat"},
				{Index: 1, Lemma: "dog", Text: "dog"},
			},
		},
	}

	var buf bytes.Buffer
	r := NewJSONRenderer(&buf)
	r.Render([]*match.SentenceMatch{sm})

	var results []match.SentenceMatch
	if err := json.Unmarshal(buf.Bytes(), &results); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].TopicName != "test-topic" {
		t.Errorf("expected topic_name 'test-topic', got %q", results[0].TopicName)
	}

	if results[0].NumExprs != 1 {
		t.Errorf("expected num_exprs 1, got %d", results[0].NumExprs)
	}

	if len(results[0].Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(results[0].Matches))
	}
}
