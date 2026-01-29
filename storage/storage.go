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

// SentenceResult represents a sentence candidate with metadata
type SentenceResult struct {
	RowID    int64
	DocID    int
	DocTitle string
	Tokens   []sent.Token
}

// DocReader defines read operations for document storage
type DocReader interface {
	// List returns the metadata (Id, Title, Labels) of all documents.
	// Content (Tokens) is not loaded.
	List() ([]sent.Doc, error)

	// Read returns a document by ID
	Read(id int) (sent.Doc, error)

	// FindCandidates returns sentence ROWIDs matching ALL given lemmas,
	// resuming after the given cursor. Returns hydrated sentences and new cursor.
	FindCandidates(lemmas []string, after Cursor, limit int) ([]SentenceResult, Cursor, error)
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
