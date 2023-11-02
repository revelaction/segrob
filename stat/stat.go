package stat

import (
	sent "github.com/revelaction/segrob/sentence"
)

type Handler struct {
	stats Stats
}

type Stats struct {
	NumSentences          int
	NumTokens             int
	TokensPerSentenceMean int
	TokensPerSentenceDis  map[int]int
}

func (h *Handler) Get() Stats {
	return h.stats
}

func NewHandler() *Handler {
	stats := Stats{TokensPerSentenceDis: map[int]int{}}
	return &Handler{
		stats: stats,
	}
}

func (h *Handler) Aggregate(doc sent.Doc) {
	h.stats.NumSentences = len(doc.Tokens)
	//
	for _, sentence := range doc.Tokens {
		h.stats.NumTokens += len(sentence)
		h.stats.TokensPerSentenceDis[len(sentence)]++
	}

	h.stats.TokensPerSentenceMean = h.stats.NumTokens / h.stats.NumSentences
}
