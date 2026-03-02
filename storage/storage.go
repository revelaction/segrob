package storage

import (
	"encoding/json"

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

// SentenceIngest represents the flat parsed structure ready for insertion.
type SentenceIngest struct {
	ID     int             `json:"id"`
	Lemmas []string        `json:"lemmas"`
	Tokens json.RawMessage `json:"tokens"` // Avoids unmarshaling tokens early
}

// DocReader defines read operations for document storage
type DocReader interface {
	// List returns document identity metadata (Id, Source).
	List() ([]sent.Meta, error)

	// Labels returns the labels for a document by ID.
	Labels(id string) ([]sent.Label, error)

	// Nlp returns sentences for a document by ID. Labels are not loaded.
	Nlp(id string) ([]sent.Sentence, error)

	// FindCandidates returns sentence candidates matching ALL given lemmas
	// AND ALL labelIDs. The caller uses ListLabels() to obtain IDs.
	FindCandidates(lemmas []string, labelIDs []int, after Cursor, limit int, onCandidate func(sent.Sentence) error) (Cursor, error)

	// ListLabels returns all labels (ID and Name). If labelSubStr is not empty,
	// only labels whose name contains the substring are returned.
	ListLabels(labelSubStr string) ([]sent.Label, error)

	// HasSentences returns true if at least one sentence exists for the given doc ID.
	HasSentences(id string) (bool, error)
}

// DocWriter defines write operations for document storage
type DocWriter interface {
	// WriteMeta persists document metadata (id, source) and its labels.
	WriteMeta(id string, source string, labels []string) error

	// WriteNLP persists sentences and updates lookup tables for the given docID.
	WriteNLP(docID string, sentences []SentenceIngest) error

	// AddLabel adds labels to a document and updates optimization tables.
	AddLabel(docID string, labels ...string) error

	// RemoveLabel removes labels from a document and updates optimization tables.
	RemoveLabel(docID string, labels ...string) error
}

// DocRepository combines read and write operations
type DocRepository interface {
	DocReader
	DocWriter
}

// CorpusMeta holds the metadata fields needed for import.
type CorpusMeta struct {
	ID     string // SHA-256 truncated hex of epub bytes
	Epub   string // epub file name (basename), used as source
	Labels string // comma-separated DC labels
}

// CorpusReader defines read operations for corpus storage
type CorpusReader interface {
	// ReadMeta retrieves id, epub, and labels for a given document ID.
	ReadMeta(id string) (CorpusMeta, error)

	// ReadTxt retrieves the txt field for a given document ID as raw bytes.
	ReadTxt(id string) ([]byte, error)
}

// CorpusRepository combines read operations
type CorpusRepository interface {
	CorpusReader
}
