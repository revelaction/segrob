package match

import (
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

	// Single scratch map for expression matching to avoid allocations
    scratchMap itemTokenMap  
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


// Reset clears the SentenceMatch for reuse while preserving allocated memory.
// It resets all fields to their zero values but keeps the underlying tokenMap
// capacity to avoid reallocation.
func (sm *SentenceMatch) Reset() {
	// Clear the map without setting it to nil - preserves allocated capacity
	clear(sm.tokenMap)
	
	// Reset all other fields to zero values
	sm.topicName = ""
	// Keep capacity, reset length to 0
	// But since MatchSentence does match.Sentence = sentence 
	// TODO pointer and reuse 
	// sm.Sentence = sm.Sentence[:0]  
	sm.Sentence = nil
	sm.NumExprs = 0
	sm.DocId = 0
	sm.SentenceId = 0
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

// MatchSentence matches a posible Topic AND a possible TopicExpr for a given sentence.
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
func (m *Matcher) MatchSentence(sentence []sent.Token, docId int, reuse *SentenceMatch) *SentenceMatch {
    hasTopic := len(m.Topic.Exprs) > 0
    hasExpr := len(m.ArgExpr) > 0
    
	// HACK: We extract the true sentence ID from the tokens themselves because
	// the current Doc structure (slice-of-slices) doesn't preserve sentence
	// metadata when passed partially.
	//
	// Identity Collisions (The "Tártaro" Bug):
	// If tokens lack metadata (SentenceId: 0), multiple matches in the same doc
	// overwrite each other in the Matcher's internal map.
	//
	// Strategy 1 (search.Sentences) now bypasses this by calling MatchSentence
	// directly and providing an authoritative ID, but other callers (like the
	// REPL) still suffer from this if they rely on Match() for aggregation.
	//
	// TODO: The proper fix is to introduce a 'Sentence' struct in the 'sentence'
	// package and update the document serialization format to:
	// type Sentence struct { Id int; Tokens []Token }
	// type Doc struct { ...; Sentences []Sentence }
    sentId := 0
    if len(sentence) > 0 {
        sentId = sentence[0].SentenceId
    }
    
    // Prepare the SentenceMatch (reuse or allocate)
    var match *SentenceMatch
    if reuse != nil {
        reuse.Reset()
        match = reuse
    } else {
        match = &SentenceMatch{tokenMap: make(itemTokenMap)}
    }
    
    // ArgExpr check
    if hasExpr {
        clear(m.scratchMap)
		// If the expr does not match, the sentence does not match
        if !sentenceExprMatch(sentence, m.ArgExpr, m.scratchMap) {
            return nil
        }
        // Copy directly to match.tokenMap
        for item, tokens := range m.scratchMap {
            match.tokenMap[item] = tokens
        }
    }
    
    // Topic expressions
    for _, expr := range m.Topic.Exprs {
        clear(m.scratchMap)
        if sentenceExprMatch(sentence, expr, m.scratchMap) {
            match.NumExprs++
            // Copy directly to match.tokenMap
            for item, tokens := range m.scratchMap {
                match.tokenMap[item] = tokens
            }
        }
    }
    
    if hasTopic && match.NumExprs == 0 {
        return nil
    }
    
    match.DocId = docId
    match.SentenceId = sentId
    match.topicName = m.Topic.Name
    match.Sentence = sentence
    
    return match
}

func sentenceExprMatch(sentence []sent.Token, expr topic.TopicExpr, mt itemTokenMap) bool {
    // mt is the scratchMap, already cleared by caller
    
    for itemIdx, item := range expr {
        switch item.Requirement() {
        case topic.RequiresOne:
            if t := matchOne(sentence, item); len(t) > 0 {
                mt[item] = t
                continue
            }
        case topic.RequiresAll:
            if matchAll(sentence, item) {
                continue
            }
        case topic.RequiresSome:
            previousItem := expr[itemIdx-1]
            if t := matchSome(sentence, mt, item, previousItem); len(t) > 0 {
                mt[item] = t
                delete(mt, previousItem)
                continue
            }
        }
        
        // Match failed
        return false
    }
    
    return true
}

func (m *Matcher) AddTopicExpr(expr topic.TopicExpr) {
	m.ArgExpr = expr
}

func NewMatcher(topic topic.Topic) *Matcher {
	return &Matcher{
		Topic: topic,
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
