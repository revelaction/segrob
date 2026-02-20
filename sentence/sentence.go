package sentence

// Sentence represents a distinct syntactic unit.
// Identity = (DocId, Id)
type Sentence struct {
	Id     int     `json:"id"`     // Sequential index in Doc (0, 1, ...)
	DocId  int     `json:"doc_id"`
	Tokens []Token `json:"tokens"`
}

type Doc struct {
	Id        int        `json:"id" toml:"-"`
	Title     string     `json:"title" toml:"title"`
	Labels    []string   `json:"labels" toml:"labels"`
	Sentences []Sentence `json:"sentences" toml:"-"`
}

// Library is a collection of Doc
type Library []Doc

// Token represents a word of the sentence, with POS and metadata.
type Token struct {
	Id         int    `json:"id"`
	Head       int    `json:"head"`
	SentenceId int    `json:"sent"`
	Pos        string `json:"pos"`
	Dep        string `json:"dep"`

	// A string containing detailed POS data
	Tag string `json:"tag"`

	// the index of the start character of the token in the original doc (set by spacy, stanza)
	Idx int `json:"idx"`

	// The unmodified word
	Text string `json:"text"`

	// The lemma of the word
	Lemma string `json:"lemma"`

	// The index of the word in the sentence, starting at 0.
	Index int `json:"index"`
}
