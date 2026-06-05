// sample/sample.go
package sample

import "github.com/revelaction/segrob/match"

// Sampler defines the contract for extracting a subset of sentence matches.
type Sampler interface {
	Sample() ([]*match.SentenceMatch, error)
}
