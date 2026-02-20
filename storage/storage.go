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
	// If labelMatch is not empty, only documents with at least one label matching the string are returned.
	// Content (Tokens) is not loaded.
	List(labelSubStr string) ([]sent.Doc, error)

	// Read returns a document by ID
	Read(id int) (sent.Doc, error)

	// FindCandidates returns sentence candidates matching ALL given lemmas AND ALL labels,
	// resuming after the given cursor. It calls onCandidate for each result.
	// Returns the new cursor and any error.
	FindCandidates(lemmas []string, labels []string, after Cursor, limit int, onCandidate func(sent.Sentence) error) (Cursor, error)

	// Labels returns all unique labels found across all documents, sorted alphabetically.
	// If labelSubStr is not empty, it returns labels that contain the labelSubStr.
	Labels(labelSubStr string) ([]string, error)
}

// DocWriter defines write operations for document storage
type DocWriter interface {
	// WriteMeta persists document metadata (Title, Labels).
	// If the document exists (by Title), it returns an error.
	WriteMeta(doc sent.Doc) error

	// WriteNLP persists sentences and updates lookup tables for the given docID.
	WriteNLP(docID int, sentences []sent.Sentence) error

	// AddLabel adds labels to a document and updates optimization tables.
	AddLabel(docID int, labels ...string) error

	// RemoveLabel removes labels from a document and updates optimization tables.
	RemoveLabel(docID int, labels ...string) error
}

// DocRepository combines read and write operations
type DocRepository interface {
	DocReader
	DocWriter
}

// Preloader defines an optional capability for repositories that require
// or support eager loading of data into memory.
type Preloader interface {
	LoadNLP(labels []string, docID *int, cb func(current, total int, name string)) error
}
