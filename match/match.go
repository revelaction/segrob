package match

import (
	"sort"
	"strings"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
)

// Matcher matchs a Doc (or a set of Docs) against a Topic (+ ArgExpr)
// A set of `Docs` can be matched by repeated `Match` calls to the Matcher.
type Matcher struct {
	Topic topic.Topic

	// ArgExpr is an additional topic expresion passed as argument to the
	// command line
	// ArgExpr have an AND semantic, they must match the sentence in addition
	// to one or more TopicExpr of the Topic.
	//
	// So if this is not empty, the match is: sentences that match one of the
	// Topic expr AND this expr.
	ArgExpr topic.TopicExpr

	// sentences contains the results of the match for each matched sentence
	// The first int is the doc Id, the second int is the sentence Id.
	//
	// Must be pointer https://stackoverflow.com/questions/32751537/why-do-i-get-a-cannot-assign-error-when-setting-value-to-a-struct-as-a-value-i
	sentences map[int]map[int]*SentenceMatch
}

// MatchedTokens is a ordered set of sentence tokens
// that are matched by a Topic Expr
//
// the following topic  expr:
//
// [{"lemma":"cuando"},{"near":3,"tag":"VERB"}],
//
// will match the following sentence:
//
// # Cuando me vio abrir los ojos, sus exclamaciones de gratitud y de júbilo provocaron tanto las risas como la
//
// the expr match two MatchedTokens:
// MatchedTokens 1) [cuando, vio]
// MatchedTokens 2) [cuando, abrir]
type matchedTokens []sent.Token

// ItemTokenMap contains the map between a topic expr token and the matched
// tokens of a sentence MatchedTokens
//
//   - The value is a slice of n 1-dimensional tokens (for unlinked
//     items), where n is the number of match ocurrences.
//   - for compound items (m items) , the values is an slice of n m-dimensional
//     slices.
type itemTokenMap map[topic.TopicExprItem][]matchedTokens

// SentenceMatch represents a sentence match of "one or more" topicExpr with a
// sentence.
// After the sentence is matched against all TopicExpr, this struct contains
// all the token matches along with the responsible Topic Items.
type SentenceMatch struct {

	// topicName is the topic that has some topic Expr which match this setence
	topicName string

	// tokenMap contains the map between the exprs items that match a sentence
	// Used mostly to highlight the words (token) that wher matched
	tokenMap itemTokenMap

	// Sentence is a slice of the sentence tokens. The output sentence is created using this.
	Sentence []sent.Token

	// NumExprs is the number of topicExpr that were matched. Used to sort the sentences
	NumExprs int

	// The doc id
	DocId int

	// SentenceId is the index of the sentence inside of the doc.
	// filled and used only to sorting and verbose output
	SentenceId int
}

func (sm *SentenceMatch) AllTokens() []sent.Token {
	sentTokens := []sent.Token{}
	for _, tokens := range sm.tokenMap {
		for _, tks := range tokens {
			sentTokens = append(sentTokens, tks...)
		}
	}

	return sentTokens
}

// Exprs returns the ExprId (string representation) of the unique TopicExpr's
// matched by the sentences.
func (sm *SentenceMatch) Exprs() []string {

	exprIds := []string{}

MATCH_TOKEN:
	for item := range sm.tokenMap {
		for _, id := range exprIds {
			if item.ExprId == id {
				continue MATCH_TOKEN
			}
		}

		exprIds = append(exprIds, item.ExprId)
	}

	return exprIds
}

func (sm *SentenceMatch) TokensForExpr(exprStr string) [][]sent.Token {

	expr2Tokens := [][]matchedTokens{}

	for item := range sm.tokenMap {
		if item.ExprId == exprStr {
			expr2Tokens = append(expr2Tokens, sm.tokenMap[item])
		}
	}

	// after each iteration of all items this provides the cartesian product
	// till now, Only the last one has the solution.
	partialResult := make([][]matchedTokens, len(expr2Tokens))

	for idx := range expr2Tokens {
		if idx == 0 {
			partialResult[idx] = expr2Tokens[0]
			continue
		}

		for _, tokens := range expr2Tokens[idx] {
			// the previous
			for _, ptTokens := range partialResult[idx-1] {
				// make new slice
				cp := matchedTokens{}
				cp = append(cp, tokens...)
				cp = append(cp, ptTokens...)

				// This is still unordered
				partialResult[idx] = append(partialResult[idx], cp)
			}
		}
	}

	res := [][]sent.Token{}
	// Convert the internal matchedTokens to external sent.Token
	for _, tokenCp := range partialResult[len(partialResult)-1] {
		c := []sent.Token{}
		for _, t := range tokenCp {
			c = append(c, t)
		}

		res = append(res, c)
	}

	return res
}

// TopicName return the topic name of the sentence match
// used by the render to prefix the topic in the `topics` command.
func (sm *SentenceMatch) TopicName() string {
	return sm.topicName
}

// Match matches a posible Topic AND a possible TopicExpr for a given Doc.
//
// Consecutive calls to Match() are possible.
//
// The semantic is as follows:
//
//   - If there are both a Topic and a TopicExpr, a sentence match only happens
//     if the TopicExpr matchs AND 'one or more' of the Topic expressions also match.
//
//   - If there is only a Topic, a sentence match only happens if 'one or more'
//     of the Topic expressions match.
//
//   - If there is only a TopicExpr, a sentence match only happens if the TopicExpr
//     matches.
func (m *Matcher) Match(doc sent.Doc) {
	hasTopic := len(m.Topic.Exprs) > 0
	hasExpr := len(m.ArgExpr) > 0
	for _, sentence := range doc.Tokens {

		// HACK: We extract the true sentence ID from the tokens themselves because
		// the current Doc structure (slice-of-slices) doesn't preserve sentence
		// metadata when passed partially.
		//
		// TODO: The proper fix is to introduce a 'Sentence' struct in the 'sentence'
		// package and update the document serialization format to:
		// type Sentence struct { Id int; Tokens []Token }
		// type Doc struct { ...; Sentences []Sentence }
		sentId := 0
		if len(sentence) > 0 {
			sentId = sentence[0].SentenceId
		}

		// We priorize the possible ArgExpr
		// If there is a ArgExpr, the sentence must match it
		argExprMap := itemTokenMap{}
		if hasExpr {
			// If the expr does not match, the sentence does not match
			if argExprMap = sentenceExprMatch(sentence, m.ArgExpr); len(argExprMap) == 0 {
				continue
			}
		}

		// initialize
		match := SentenceMatch{tokenMap: itemTokenMap{}}
		for _, expr := range m.Topic.Exprs {

			// OR semantic: one of the Topic expressions  must match
			if topicMatchMap := sentenceExprMatch(sentence, expr); len(topicMatchMap) > 0 {
				match.NumExprs++
				for item, tokens := range topicMatchMap {
					match.tokenMap[item] = tokens
				}
			}
		}

		if hasTopic {
			if match.NumExprs == 0 {
				continue
			}
		}

		// If we are here, there is one or more topic expression matches and maybe expr matches
		if hasExpr {
			// we have also expr match, add to the match
			for item, tokens := range argExprMap {
				match.tokenMap[item] = tokens
			}
		}

		if _, ok := m.sentences[doc.Id]; !ok {
			m.sentences[doc.Id] = map[int]*SentenceMatch{}
		}

		match.DocId = doc.Id
		match.SentenceId = sentId
		match.topicName = m.Topic.Name
		match.Sentence = sentence
		m.sentences[doc.Id][sentId] = &match
	}
}

func sentenceExprMatch(sentence []sent.Token, expr topic.TopicExpr) itemTokenMap {

	mt := itemTokenMap{}
MATCH_TOKEN:
	for itemIdx, item := range expr {
		switch item.Requirement() {
		case topic.RequiresOne:

			if t := matchOne(sentence, item); len(t) > 0 {
				mt[item] = t
				continue MATCH_TOKEN
			}
		case topic.RequiresAll:
			if matchAll(sentence, item) {
				continue MATCH_TOKEN
			}
		case topic.RequiresSome:

			previousItem := expr[itemIdx-1]
			if t := matchSome(sentence, mt, item, previousItem); len(t) > 0 {
				mt[item] = t
				delete(mt, previousItem)
				continue MATCH_TOKEN
			}
		}

		// we are here because some topic epr item token was not matched
		return itemTokenMap{}
	}

	return mt
}

func (m *Matcher) AddTopicExpr(expr topic.TopicExpr) {
	m.ArgExpr = expr
}

// Sentences is the slice of matched sentences SentenceMatch.
// The SentenceMatch's are ordered (sentences with more expr matches of the
// topics comes first)
func (m *Matcher) Sentences() []*SentenceMatch {
	// flatten the results
	resultSlice := []*SentenceMatch{}
	for _, docResults := range m.sentences {
		for _, sent := range docResults {
			resultSlice = append(resultSlice, sent)
		}
	}

	// Sort by Counter
	return sortByNumMatches(resultSlice)
}

func NewMatcher(topic topic.Topic) *Matcher {
	return &Matcher{
		Topic:     topic,
		sentences: map[int]map[int]*SentenceMatch{},
	}
}

func matchOne(sentence []sent.Token, item topic.TopicExprItem) (matched []matchedTokens) {
	for _, t := range sentence {
		if isTokenMatch(t, item) {
			matched = append(matched, matchedTokens{t})
		}
	}

	return matched
}

func matchAll(sentence []sent.Token, item topic.TopicExprItem) bool {
	for _, t := range sentence {
		if isTokenMatch(t, item) {
			continue
		}

		return false
	}

	return true
}

func matchSome(sentence []sent.Token, mt itemTokenMap, item, previousItem topic.TopicExprItem) (matched []matchedTokens) {

	previousItemCandidates := mt[previousItem]

	sentenceEnd := len(sentence) - 1

	for _, tc := range previousItemCandidates {
		var subSentence []sent.Token
		previousTokenIndex := tc[len(tc)-1].Index
		// Check If previous is the last token of sentence, there is no
		// possibility of a near match
		if sentenceEnd == previousTokenIndex {
			continue
		}

		requiredEnd := previousTokenIndex + int(item.Near)
		if requiredEnd > sentenceEnd {
			requiredEnd = sentenceEnd
		}

		subSentence = sentence[previousTokenIndex+1 : requiredEnd+1]
		for _, t := range subSentence {
			if isTokenMatch(t, item) {
				newCandidate := []sent.Token{}
				newCandidate = append(newCandidate, tc...)
				newCandidate = append(newCandidate, t)
				matched = append(matched, newCandidate)
			}
		}
	}

	// no new candidate
	return
}

func isTokenMatch(t sent.Token, item topic.TopicExprItem) bool {
	//
	// Lemma field
	//
	if len(item.Lemma) > 0 {
		if strings.HasPrefix(item.Lemma, "!") {
			if strings.TrimPrefix(item.Lemma, "!") == t.Lemma {
				return false
			}
		} else {
			// optimistically try to split possible OR values
			// If no "|" just one value
			isOrValue := false
			for _, orValue := range strings.Split(item.Lemma, "|") {
				if orValue == t.Lemma {
					isOrValue = true
					break
				}
			}

			if !isOrValue {
				return false
			}
		}
	}

	//
	// Tag field
	//
	// A Tag string from spacy contains substring seaprated with '|':
	//      DET__Definite=Def|Gender=Fem|Number=Sing|PronType=Art
	// Do not mistake with our | operator.
	if len(item.Tag) > 0 {
		switch separator(item.Tag) {
		case "|":
			// OR
			isMatched := false
			for _, orItem := range strings.Split(item.Tag, "|") {
				if strings.Contains(t.Tag, orItem) {
					isMatched = true
					break
				}
			}

			if !isMatched {
				return false
			}
		case "+":
			// AND: must contain each
			for _, andItem := range strings.Split(item.Tag, "+") {
				if !strings.Contains(t.Tag, andItem) {
					return false
				}
			}
		default:
			// single
			if !strings.Contains(t.Tag, item.Tag) {
				return false
			}
		}
	}

	// remove use Tag instead
	if len(item.Pos) > 0 {
		if item.Pos != t.Pos {
			return false
		}
	}

	return true
}

func separator(field string) string {
	if strings.Contains(field, "|") {
		return "|"
	}

	if strings.Contains(field, "+") {
		return "+"
	}

	return ""
}

func sortByNumMatches(resultSlice []*SentenceMatch) []*SentenceMatch {
	sort.Slice(resultSlice, func(i, j int) bool {

		// Counter sorting
		if resultSlice[i].NumExprs > resultSlice[j].NumExprs {
			return true
		}

		if resultSlice[i].NumExprs < resultSlice[j].NumExprs {
			return false
		}

		// docId sorting
		if resultSlice[i].DocId < resultSlice[j].DocId {
			return true
		}
		if resultSlice[i].DocId > resultSlice[j].DocId {
			return false
		}

		//sentenceId sorting
		if resultSlice[i].SentenceId < resultSlice[j].SentenceId {
			return true
		}

		return false
	})

	return resultSlice
}
