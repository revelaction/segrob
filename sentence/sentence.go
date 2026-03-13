package sentence

// Sentence represents a distinct syntactic unit.
// Identity = (DocId, Id)
type Sentence struct {
	SentenceId int     `json:"id"` // Sequential index in Doc (0, 1, ...)
	DocId      string  `json:"doc_id"`
	Tokens     []Token `json:"tokens"`
}

// Labels maps a label's Name to its integer ID.
type Labels map[string]int

// Reverse returns a map from ID to Name.
func (l Labels) Reverse() map[int]string {
	rev := make(map[int]string, len(l))
	for name, id := range l {
		rev[id] = name
	}
	return rev
}

type Meta struct {
	Id       string `json:"id" toml:"-"`
	Source   string `json:"source" toml:"source"`
	LabelIDs []int  `json:"label_ids"`
}

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
