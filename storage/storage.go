package storage

import (
	"encoding/json"
	"time"

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

	// HasLabelsOptimization returns true if at least one sentence_labels row exists for the given doc ID.
	HasLabelsOptimization(id string) (bool, error)

	// HasLemmaOptimization returns true if at least one sentence_lemmas row exists for the given doc ID.
	HasLemmaOptimization(id string) (bool, error)

	// Exists returns true if a document with the given ID is present in the docs table.
	Exists(id string) (bool, error)
}

// DocWriter defines write operations for document storage
type DocWriter interface {
	// WriteMeta persists document metadata (id, source) and its labels.
	WriteMeta(id string, source string, labels []string) ([]int, error)

	// WriteNlpData persists sentences for the given docID.
	WriteNlpData(docID string, sentences []SentenceIngest) error

	// WriteLabelsOptimization writes sentence_labels rows for the given docID.
	WriteLabelsOptimization(docID string, labelIDs []int) error

	// WriteLemmaOptimization writes sentence_lemmas rows for the given docID.
	WriteLemmaOptimization(docID string, sentences []SentenceIngest) error

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
	// List returns records (ID, Labels, txt_hash, nlp_created_at, etc.).
	List() ([]CorpusRecord, error)

	// ReadMeta retrieves id, epub, and labels for a given document ID.
	ReadMeta(id string) (CorpusMeta, error)

	// ReadTxt retrieves the txt field for a given document ID as raw bytes.
	ReadTxt(id string) ([]byte, error)

	// ReadNlp retrieves the raw NLP JSON payload for a given document ID.
	ReadNlp(id string) ([]byte, error)

	// Exists returns true if a record with the given ID is present in the docs table.
	Exists(id string) (bool, error)
}

// CorpusRecord holds all data collected for a single epub that will be
// inserted as one row in the corpus docs table.
type CorpusRecord struct {
	CorpusMeta
	Txt          string // full plain text from pandoc
	TxtHash      string // SHA-256 hex of txt bytes
	TxtCreatedAt time.Time
	TxtEdit      bool
	TxtEditAt    time.Time
	TxtEditBy    string
	TxtEditNotes string
	TxtAck       bool
	TxtAckAt     time.Time
	TxtAckBy     string
	NlpCreatedAt time.Time
	NlpAck       bool
	NlpAckAt     time.Time
	NlpAckBy     string
	DeletedAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TimeParse parses a RFC3339 string into a time.Time.
// This should be used when reading timestamps from SQLite to convert them
// back to time.Time values. Returns an error if the input string is not
// in RFC3339 format.
func TimeParse(s string) (time.Time, error) {
	// Handle empty strings gracefully, returning zero time and no error,
	// as some DB fields might be nullable/empty timestamps.
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

// HasTxt reports whether plain-text content has been generated and stored.
// The Txt field is not populated by most queries due to its size — TxtHash
// is the authoritative signal for text presence.
func (r CorpusRecord) HasTxt() bool {
	return r.TxtHash != ""
}

// HasNlp reports whether Nlp content has been generated and stored.
// The Nlp field is not even in the CorpusRecord — NlpCreatedAt
// is the authoritative signal for Nlp presence.
func (r CorpusRecord) HasNlp() bool {
	return !r.NlpCreatedAt.IsZero()
}

func (r CorpusRecord) HasAck() bool {
	return r.TxtAck && r.NlpAck
}

// CorpusWriter defines write operations for corpus storage
type CorpusWriter interface {
	// WriteStream inserts corpus records yielded by the iterator.
	WriteStream(seq func(yield func(CorpusRecord, error) bool)) error

	// WriteNlp stores the NLP JSON payload for the given document ID.
	WriteNlp(id string, nlp []byte) error

	// ClearNlp sets the nlp field to NULL for the given document ID.
	ClearNlp(id string) error

	// UpdateTxt updates the txt field and its associated metadata for the given document ID.
	UpdateTxt(id string, txt []byte, txtHash string, by string, notes string) error

	// AckTxt updates the txt_ack fields for the given document ID.
	AckTxt(id string, by string) error

	// AckNlp updates the nlp_ack fields for the given document ID.
	AckNlp(id string, by string) error

	// Delete removes a document from the corpus by its ID.
	Delete(id string) error
}

// CorpusRepository combines read and write operations
type CorpusRepository interface {
	CorpusReader
	CorpusWriter
}
