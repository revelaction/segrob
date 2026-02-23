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

func (h *Handler) Aggregate(sentences []sent.Sentence) {
	h.stats.NumSentences = len(sentences)
	if h.stats.NumSentences == 0 {
		return
	}
	//
	for _, sentence := range sentences {
		h.stats.NumTokens += len(sentence.Tokens)
		h.stats.TokensPerSentenceDis[len(sentence.Tokens)]++
	}

	h.stats.TokensPerSentenceMean = h.stats.NumTokens / h.stats.NumSentences
}
