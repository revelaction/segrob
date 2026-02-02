package topic

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

const (
	RequiresOne = iota
	RequiresSome
	RequiresAll
)

type Topic struct {

	// the topic name
	Name string

	// the expression of the topic
	Exprs []TopicExpr
}

type TopicExpr []TopicExprItem

func (m TopicExpr) String() string {
	sl := []string{}
	for _, item := range m {
		if item.Near > 0 {
			sl = append(sl, strconv.Itoa(item.Near))
		}

		if len(item.Lemma) > 0 {
			sl = append(sl, item.Lemma)
			continue
		}

		if len(item.Tag) > 0 {
			sl = append(sl, item.Tag)
		}
	}

	return strings.Join(sl, " ")
}

// Lemmas returns all unique non-negative lemmas present in the TopicExpr.
// Negative lemmas (starting with '!') are excluded because they cannot be used
// for indexed candidate retrieval in storage; they are handled later by the Matcher.
func (m TopicExpr) Lemmas() []string {
	seen := make(map[string]bool)
	var lemmas []string
	for _, item := range m {
		if item.Lemma != "" && !strings.HasPrefix(item.Lemma, "!") {
			if !seen[item.Lemma] {
				seen[item.Lemma] = true
				lemmas = append(lemmas, item.Lemma)
			}
		}
	}
	return lemmas
}

// LemmaSets returns a slice of lemma sets, one for each expression in the topic.
// It only includes positive lemmas suitable for indexed searching.
func (t Topic) LemmaSets() [][]string {
	var sets [][]string
	for _, e := range t.Exprs {
		lemmas := e.Lemmas()
		if len(lemmas) > 0 {
			sets = append(sets, lemmas)
		}
	}
	return sets
}

type TopicExprItem struct {

	// The Expr index.
	ExprIndex int `json:"-"`

	// ExprId is the Expresion String(). It should be unique as two identical expresions can
	// not be in the same topic file.
	ExprId string `json:"-"`

	// TopicName references the Topic of the Item
	TopicName string `json:"-"`

	Near  int    `json:"near,omitempty"`
	Lemma string `json:"lemma,omitempty"`
	Pos   string `json:"pos,omitempty"`
	Dep   string `json:"dep,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

// Library is a collection of topics
type Library []Topic

// Names returns a list of all topic names in the library
func (l Library) Names() []string {
	var names []string
	for _, t := range l {
		names = append(names, t.Name)
	}
	return names
}

func (m TopicExprItem) Requirement() int {
	if m.Near > 0 {
		return RequiresSome
	}

	if strings.HasPrefix(m.Lemma, "!") {
		return RequiresSome
	}

	return RequiresOne
}

// Parse parses the user input and converts to a TopicExpr.
func Parse(args []string) (TopicExpr, error) {

	isLastInt := false
	var expr TopicExpr
	var lastNear int64 = 0
	for idx, arg := range args {
		near, err := strconv.ParseInt(arg, 10, 64)
		if err == nil {
			if idx == 0 {
				return nil, errors.New("First expression argument can not be number")
			}

			if isLastInt {
				return nil, errors.New("Can not parse two consecutive numbers in the expression")
			}

			lastNear = near
			isLastInt = true
			continue
		}

		firstChar := []rune(arg)[0]

		category := "lemma"
		if unicode.IsUpper(firstChar) && unicode.IsLetter(firstChar) {
			category = "tag"
		}

		switch category {
		case "tag":
			expr = append(expr, TopicExprItem{Tag: arg, Near: int(lastNear)})
		default:
			expr = append(expr, TopicExprItem{Lemma: arg, Near: int(lastNear)})
		}

		lastNear = 0
		isLastInt = false
	}

	return expr, nil
}

// EqualExpr determines if two expresions are the same.
// the Equality requires slice order. It does not support conmutativity:
//
//	itemA, itemB != itemB, itemA
func EqualExpr(a, b TopicExpr) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if !EqualExprItem(v, b[i]) {
			return false
		}
	}
	return true
}

// EqualExprItem determines if two expresions items are the same. Two
// TopicExprItem are the same if they have the same Lemma, Tag, Near, Dep and
// Pos fields.
func EqualExprItem(a, b TopicExprItem) bool {

	if a.Lemma != b.Lemma {
		return false
	}

	if a.Near != b.Near {
		return false
	}

	if a.Tag != b.Tag {
		return false
	}

	if a.Pos != b.Pos {
		return false
	}

	if a.Dep != b.Dep {
		return false
	}

	return true
}
