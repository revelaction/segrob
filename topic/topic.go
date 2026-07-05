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
)

type Topic struct {
	Name  string      `json:"name"`
	Exprs []TopicExpr `json:"exprs"`
}

type TopicExpr struct {
	Items   []TopicExprItem `json:"items"`
	Flagged bool            `json:"flagged"`
}

func (m TopicExpr) String() string {
	sl := []string{}
	for _, item := range m.Items {
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

// Lemmas returns all unique lemmas present in the TopicExpr.
func (m TopicExpr) Lemmas() []string {
	seen := make(map[string]bool)
	var lemmas []string
	for _, item := range m.Items {
		if item.Lemma != "" {
			if !seen[item.Lemma] {
				seen[item.Lemma] = true
				lemmas = append(lemmas, item.Lemma)
			}
		}
	}
	return lemmas
}

type TopicExprItem struct {
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

// Parse parses the user input and converts to a TopicExpr.
func Parse(args []string) (TopicExpr, error) {

	isLastInt := false
	var items []TopicExprItem
	var lastNear int64 = 0
	for idx, arg := range args {
		near, err := strconv.ParseInt(arg, 10, 64)
		if err == nil {
			if idx == 0 {
				return TopicExpr{}, errors.New("first expression argument can not be number")
			}

			if isLastInt {
				return TopicExpr{}, errors.New("can not parse two consecutive numbers in the expression")
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
			items = append(items, TopicExprItem{Tag: arg, Near: int(lastNear)})
		default:
			items = append(items, TopicExprItem{Lemma: arg, Near: int(lastNear)})
		}

		lastNear = 0
		isLastInt = false
	}

	return TopicExpr{Items: items}, nil
}

// EqualExpr determines if two expresions are the same.
// the Equality requires slice order. It does not support conmutativity:
//
//	itemA, itemB != itemB, itemA
func EqualExpr(a, b TopicExpr) bool {
	if len(a.Items) != len(b.Items) {
		return false
	}

	for i, v := range a.Items {
		if !EqualExprItem(v, b.Items[i]) {
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

// exprKey builds a canonical, order-sensitive string representation of an
// expression's items, suitable for use as a map key. It covers all five fields
// compared by EqualExprItem (Lemma, Near, Tag, Pos, Dep) in fixed order, with
// \x00 separating fields and \x01 marking item boundaries. Flagged is excluded
// to match EqualExpr's equality semantics.
func exprKey(e TopicExpr) string {
	var sb strings.Builder
	for _, item := range e.Items {
		sb.WriteString(item.Lemma)
		sb.WriteByte(0)
		sb.WriteString(strconv.Itoa(item.Near))
		sb.WriteByte(0)
		sb.WriteString(item.Tag)
		sb.WriteByte(0)
		sb.WriteString(item.Pos)
		sb.WriteByte(0)
		sb.WriteString(item.Dep)
		sb.WriteByte(1) // item boundary
	}
	return sb.String()
}

// Deduplicate removes duplicate expressions from a slice, preserving order.
// Two expressions are considered equal if EqualExpr returns true.
// Flagged is ignored for equality purposes.
//
// Complexity: O(n·m) where n = len(exprs) and m = average items per expression.
// Each expression is keyed once (O(m) per key), then looked up in the map (O(1)
// amortized). This beats the O(n²·m) nested-loop approach for any non-trivial n.
func Deduplicate(exprs []TopicExpr) []TopicExpr {
	seen := make(map[string]bool, len(exprs))
	result := make([]TopicExpr, 0, len(exprs))
	for _, e := range exprs {
		key := exprKey(e)
		if !seen[key] {
			seen[key] = true
			result = append(result, e)
		}
	}
	return result
}
