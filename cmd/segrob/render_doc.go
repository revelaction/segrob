// cmd/segrob/render_doc.go
package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
)

func renderDoc(sentences []sent.Sentence, opts ShowOptions, ui UI) {
	start := opts.Start
	if start < 0 {
		start = 0
	}
	if start >= len(sentences) {
		return
	}

	subset := sentences[start:]
	if opts.Count != nil {
		limit := *opts.Count
		if limit < len(subset) {
			subset = subset[:limit]
		}
	}

	r := render.NewCLIRenderer()
	r.HasColor = false
	for i, sentence := range subset {
		prefix := fmt.Sprintf("✍  %d ", start+i)
		r.Sentence(sentence.Tokens, prefix)
	}
}
