package render

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/revelaction/segrob/match"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

const (
	partialOffset = 6
	Defaultformat = "all"
)

var (
	Black   = "\033[1;30m"
	Red     = "\033[1;31m"
	Green   = "\033[1;32m"
	Yellow  = "\033[0;33m"
	Purple  = "\033[1;34m"
	Magenta = "\033[1;35m"
	Teal    = "\033[1;36m"
	Gray    = "\033[0;37m"
	White   = "\033[1;37m"
	Off     = "\033[0m"
	//Yellow256  = "\033[1;38;5;202m"
	Yellow256 = "\033[1;38;5;130m"
	Grey256   = "\033[1;38;5;145m"
	Green256  = "\033[1;38;5;70m"
	ClearLine = "\033[K"
)

func SupportedFormats() []string {
	return []string{"all", "part", "lemma", "aggr"}
}

type Renderer struct {
	HasColor bool

	HasPrefix bool

	PrefixDocFunc   func(*match.SentenceMatch) string
	PrefixTopicFunc func(*match.SentenceMatch) string

	// Format determines the format of the sentence
	//
	// all: print all sentence
	// part: print the sorrounding of the matches in the sentence, cut the rest.
	// matches: print only matched words of the sentence
	Format string

	// Show only sentences with this amount of matches
	NumMatches int

	DocNames map[int]string
}

// Match matches the doc with the current TopicExpr and fills the Matches
// property with the matched sentences.
func (r *Renderer) Match(resultsSorted []*match.SentenceMatch) {

	// if aggr format, we collect the aggr lemmas here
	aggregatedLemmas := map[string]int{}

	for _, sentenceMatch := range resultsSorted {
		if r.NumMatches > 0 && sentenceMatch.NumExprs < r.NumMatches {
			break
		}

		sentTokens := sentenceMatch.AllTokens()

		prefixDoc := r.buildPrefixDoc(sentenceMatch)
		prefixTopic := r.buildPrefixTopic(sentenceMatch)

		var text string
		switch r.Format {
		case "all":
			text = r.sentence(sentenceMatch.Sentence, sentTokens)
		case "part":
			text = r.syntagma(sentenceMatch.Sentence, sentTokens)

		case "lemma":
			text = r.lemma(sentTokens)
		case "aggr":
			r.aggregateLemma(sentTokens, aggregatedLemmas)

			continue
		}

		fmt.Fprintf(os.Stdout, "%s%s%s\n", prefixDoc, prefixTopic, strings.ReplaceAll(text, "\n", " "))
	}

	if r.Format == "aggr" {
		r.aggrLemmas(aggregatedLemmas)
	}
}

func (r *Renderer) AddDocName(docId int, name string) {
	r.DocNames[docId] = name
}

func NewRenderer() *Renderer {
	return &Renderer{DocNames: map[int]string{}}
}

func (r *Renderer) Sentence(s []sent.Token, prefix string) {
	text := r.sentence(s, []sent.Token{})
	fmt.Fprintf(os.Stdout, "%s%s\n", prefix, strings.ReplaceAll(text, "\n", " "))
}

func (r *Renderer) SentenceString(s []sent.Token, matches []sent.Token) string {
	text := r.sentence(s, matches)
	return strings.ReplaceAll(text, "\n", " ")
}

// SentenceBlindedString returns the original text of the sentence s with the
// words in matches substituted by a mask (f.ex. XXX)
func (r *Renderer) SentenceBlindedString(s []sent.Token, matches []sent.Token) string {

	blinded := []sent.Token{}
	for _, t := range s {
		for _, mt := range matches {
			if t.Index == mt.Index {
				l := len([]rune(t.Text))
				t.Text = strings.Repeat("#", l)
				blinded = append(blinded, t)
				break
			}
		}

		blinded = append(blinded, t)
	}

	text := r.sentence(blinded, nil)
	rText := strings.ReplaceAll(text, "\n", " ")
	re := regexp.MustCompile(`#+`)
	return re.ReplaceAllLiteralString(rText, "###")
}

func (r *Renderer) sentence(sentence, matches []sent.Token) string {
	var str strings.Builder
	var lastIdx, lastLen int
	for _, token := range sentence {
		l := len([]rune(token.Text))
		if lastIdx == 0 {
			str.WriteString(colorToken(token, matches, r.HasColor))
			lastIdx = token.Idx
			lastLen = l
			continue
		}

		// in the segrob format, both (or more) parts of the multi token word
		// have the same `text` field, and the same `idx`:
		//       {
		//         "id": 455,
		//         "pos": "VERB",
		//         "tag": "VerbForm=Inf",
		//         "dep": "xcomp",
		//         "head": 3,
		//         "text": "envolverse",
		//         "sent": 0,
		//         "idx": 2431,
		//         "index": 4,
		//         "lemma": "envolver"
		//       },
		//       {
		//         "id": 456,
		//         "pos": "PRON",
		//         "tag": "Case=Acc,Dat|Person=3|PrepCase=Npr|PronType=Prs|Reflex=Yes",
		//         "dep": "obj",
		//         "head": 4,
		//         "text": "envolverse",
		//         "sent": 0,
		//         "idx": 2431,
		//         "index": 5,
		//         "lemma": "él"
		//       },
		//
		// the `idx` field is the offset of the sentence in the original txt source (rune, utf8 based).
		// By having both token the same idx, we avoid the rendering of the token text again.
		// 2431 -2431 = 0
		diff := token.Idx - lastIdx

		if diff > 0 {
			str.WriteString(strings.Repeat(" ", diff-lastLen))
			str.WriteString(colorToken(token, matches, r.HasColor))
		}

		lastIdx = token.Idx
		lastLen = l
	}

	return str.String()
}

func (r *Renderer) syntagma(sentence, matches []sent.Token) string {
	// if not matches, we print the whole sentence
	if len(matches) == 0 {
		return r.sentence(sentence, matches)
	}

	// Get the first match
	//
	// matches slice is at least 1, take the first as the first candidate
	firstMatchIndex := matches[0].Index
	for _, mt := range matches {
		if mt.Index < firstMatchIndex {
			firstMatchIndex = mt.Index
		}
	}

	// Get the last match
	// matches slice is at least 1, take the first as the first candidate
	lastMatchIndex := matches[0].Index
	for _, mt := range matches {
		if mt.Index > lastMatchIndex {
			lastMatchIndex = mt.Index
		}
	}

	lastTokenIndex := len(sentence) - 1

	syntagmaFirstIdx := 0
	syntagmaLastIdx := lastTokenIndex

	// if firstMatchIndex less, show from start sentence
	if firstMatchIndex > partialOffset {
		// can be less with multi token word
		syntagmaFirstIdx = firstMatchIndex - partialOffset
	}

	if lastTokenIndex-lastMatchIndex > partialOffset {
		syntagmaLastIdx = lastMatchIndex + partialOffset
	}

	return r.sentence(sentence[syntagmaFirstIdx:syntagmaLastIdx+1], matches)
}

// Topic renders topic expressions in a mode compatible with the topic parser
// We only support render of topic expr Items compatble with the parser.
// This `in topic file` created item
//
//	[{"lemma":"tomar"}, {"near": 2, "Tag":"NOUN","lemma":"mano"}],
//
// will be rendered as:
//
//	tomar 2 mano
//
// thus priorizing lemma to Tag fields.
func (r *Renderer) Topic(exprs []topic.TopicExpr) {
	prefix := ""
	for _, expr := range exprs {
		exprSlice := []string{}

		for _, item := range expr {
			if item.Near > 0 {
				exprSlice = append(exprSlice, strconv.Itoa(item.Near))
			}

			if item.Lemma != "" {
				exprSlice = append(exprSlice, item.Lemma)
				// Lemma priorization
				continue
			}

			if item.Tag != "" {
				exprSlice = append(exprSlice, fmt.Sprintf("%q", item.Tag))
			}
		}

		fmt.Fprintf(os.Stdout, "%s%s\n", prefix, strings.Join(exprSlice, " "))
	}
}

func (r *Renderer) LemmaString(s []sent.Token, matches []sent.Token) string {
	return r.lemma(matches)
}

// lemma renders only the matched tokens (the lemma field)
func (r *Renderer) lemma(matches []sent.Token) string {
	matchedWords := []string{}
	for _, t := range matches {
		matchedWords = append(matchedWords, t.Lemma)
	}

	return strings.Join(matchedWords, " ")
}

func (r *Renderer) aggregateLemma(matches []sent.Token, aggrLemmas map[string]int) {
	matchedTokens := []sent.Token{}

OUTER:
	for _, t := range matches {
		// avoid duplicates word  (same index in sentence)
		for _, m := range matchedTokens {
			if m.Index == t.Index {
				continue OUTER
			}
		}

		matchedTokens = append(matchedTokens, t)
	}

	// build concatenated lemma
	matchedLemmas := []string{}
	for _, m := range matchedTokens {
		matchedLemmas = append(matchedLemmas, m.Lemma)
	}

	aggrLemmas[strings.Join(matchedLemmas, " ")]++
}

func colorToken(token sent.Token, matches []sent.Token, hasColor bool) string {
	if !hasColor {
		return token.Text
	}

	for _, mt := range matches {
		if mt.Id == token.Id {
			return Green256 + token.Text + Off
		}
	}

	return token.Text
}

func (r *Renderer) buildPrefixDoc(sentenceMatch *match.SentenceMatch) string {

	if !r.HasPrefix {
		return PrefixFuncEmpty(sentenceMatch)
	}

	if r.PrefixDocFunc != nil {
		return r.PrefixDocFunc(sentenceMatch)
	}

	// Default
	return fmt.Sprintf("[%37s %2d %5d:%2d] ✍  ", r.title(sentenceMatch.Sentence.DocId), sentenceMatch.Sentence.DocId, sentenceMatch.Sentence.Id, sentenceMatch.NumExprs)
}

func PrefixFuncEmpty(sentenceMatch *match.SentenceMatch) string {
	return ""
}

func PrefixFuncIconHand(sentenceMatch *match.SentenceMatch) string {
	return fmt.Sprintf("%2d ✍  ", sentenceMatch.Sentence.Id)
}

func PrefixFuncIconLabel(sentenceMatch *match.SentenceMatch) string {
	return fmt.Sprintf("%2d 🔖 ", sentenceMatch.Sentence.Id)

}

func (r *Renderer) buildPrefixTopic(sm *match.SentenceMatch) string {

	if !r.HasPrefix {
		return PrefixFuncEmpty(sm)
	}

	if r.PrefixTopicFunc != nil {
		return r.PrefixTopicFunc(sm)
	}

	topicName := sm.TopicName()

	topicPrefix := "🏷  " + Yellow256 + topicName + Off
	return fmt.Sprintf("[%-50s] ✍  ", topicPrefix)
}

func (r *Renderer) title(docId int) string {
	title := r.DocNames[docId]
	l := len(title)
	var part string
	if l <= 20 {
		part = fmt.Sprintf("%-20s", title)
	} else {
		part = title[:20]
	}

	return Grey256 + part + Off
}

// NextFormat sets the Renderer Format option to a different one, following
// the SupportedFormats() order.
func (r *Renderer) NextFormat() {

	supported := SupportedFormats()
	for i, format := range supported {
		if format == r.Format {
			switch i {
			case len(supported) - 1:
				r.Format = supported[0]
			default:
				r.Format = supported[i+1]
			}

			break
		}
	}
}

func (r *Renderer) NextPrefix() {

	// toggle
	r.HasPrefix = !r.HasPrefix
}

func (r *Renderer) aggrLemmas(agls map[string]int) {
	// flatten map to use sortSlice
	sl := []struct {
		NumSent  int
		LemmaStr string
	}{}

	for lemmaStr, index := range agls {
		sl = append(sl, struct {
			NumSent  int
			LemmaStr string
		}{index, lemmaStr})
	}

	// Sort
	sort.SliceStable(sl, func(i, j int) bool {

		// first by num sentences
		if sl[i].NumSent > sl[j].NumSent {
			return true
		}

		if sl[i].NumSent < sl[j].NumSent {
			return false
		}

		// len of lemmas string
		if len(sl[i].LemmaStr) < len(sl[j].LemmaStr) {
			return true
		}

		if len(sl[i].LemmaStr) > len(sl[j].LemmaStr) {
			return false
		}

		return false
	})

	var prefix string
	for _, s := range sl {
		if r.HasPrefix {
			prefix = fmt.Sprintf("[%5d] ✍  ", s.NumSent)
		}

		fmt.Fprintf(os.Stdout, "%s%s\n", prefix, s.LemmaStr)
	}
}
