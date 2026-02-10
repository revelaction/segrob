package storage

import (
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

// TopicReader defines read operations for topic storage
type TopicReader interface {
	// ReadAll returns all topics from storage
	ReadAll() (topic.Library, error)

	// Read returns a single topic by name
	Read(name string) (topic.Topic, error)
}

// TopicWriter defines write operations for topic storage
type TopicWriter interface {
	// Write persists a topic to storage
	Write(tp topic.Topic) error
}

// TopicRepository combines read and write operations
type TopicRepository interface {
	TopicReader
	TopicWriter
}

// Cursor for paginated lemma-based queries
type Cursor int64

// DocReader defines read operations for document storage
type DocReader interface {
	// List returns the metadata (Id, Title, Labels) of documents.
	// If labelMatch is not empty, only documents with at least one label containing the string are returned.
	// Content (Tokens) is not loaded.
	List(labelMatch string) ([]sent.Doc, error)

	// Read returns a document by ID
	Read(id int) (sent.Doc, error)

	// FindCandidates returns sentence candidates matching ALL given lemmas AND ALL labels,
	// resuming after the given cursor. It calls onCandidate for each result.
	// Returns the new cursor and any error.
	FindCandidates(lemmas []string, labels []string, after Cursor, limit int, onCandidate func(sent.Sentence) error) (Cursor, error)

	// Labels returns all unique labels found across all documents, sorted alphabetically.
	// If pattern is not empty, it returns labels that contain the pattern.
	Labels(pattern string) ([]string, error)
}

// DocWriter defines write operations for document storage
type DocWriter interface {
	// Write persists a document and its sentences/lemmas to storage
	Write(doc sent.Doc) error
}

// DocRepository combines read and write operations
type DocRepository interface {
	DocReader
	DocWriter
}

// Preloader defines an optional capability for repositories that require
// or support eager loading of data into memory.
type Preloader interface {
	Preload(labels []string, cb func(current, total int, name string)) error
}
