package render

import (
	"encoding/json"
	"io"

	"github.com/revelaction/segrob/match"
)

// JSONRenderer writes SentenceMatch results as JSON to a writer.
type JSONRenderer struct {
	W io.Writer
}

// NewJSONRenderer creates a JSONRenderer writing to w.
func NewJSONRenderer(w io.Writer) *JSONRenderer {
	return &JSONRenderer{W: w}
}

// Render serializes sentence match results as a JSON array.
func (r *JSONRenderer) Render(results []*match.SentenceMatch) {
	json.NewEncoder(r.W).Encode(results)
}

// compile-time interface check
var _ Renderer = (*JSONRenderer)(nil)
