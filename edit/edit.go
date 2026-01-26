package edit

import (
	"errors"
	"fmt"
	"github.com/revelaction/segrob/topic"
	"strings"

	prompt "github.com/c-bata/go-prompt"
)

const (
	actionAdd    = 1
	actionDelete = 0
)

type Handler struct {
	Library topic.Library

	TopicReader topic.TopicReader
	TopicWriter topic.TopicWriter
}

func NewHandler(l topic.Library, r topic.TopicReader, w topic.TopicWriter) *Handler {
	return &Handler{
		Library:     l,
		TopicReader: r,
		TopicWriter: w,
	}
}

func (h *Handler) Run() error {

	fmt.Println("🔑 Ctrl+L: clear, 🔧 quit")

	// initialize prompt history
	history := []string{}

	for {

		// PromtForEdit
		in := prompt.Input("      🔖 ", h.completer(),
			prompt.OptionTitle("segrob edit"),
			prompt.OptionPrefixTextColor(prompt.Yellow),
			prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
			prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
			prompt.OptionMaxSuggestion(12),
			prompt.OptionHistory(history),
		)

		if in == "quit" {
			return nil
		}

		history = append(history, in)
		tp, expr, action, err := h.parse(in)
		if err != nil {
			fmt.Printf("❌ %s\n", err)
			continue
		}

		if action == actionAdd {
			if exprExistInTopic(tp, expr) {
				fmt.Printf("❌ %s\n", "Expression already exist.")
				continue
			}

			tp.Exprs = append(tp.Exprs, expr)

		} else {

			if !exprExistInTopic(tp, expr) {
				fmt.Printf("❌ %s\n", "Expression des not exist.")
				continue
			}

			tp = removeExprFromTopic(tp, expr)
		}

		werr := h.TopicWriter.Write(tp)
		if werr != nil {
			return werr
		}

		// reload the topic after write
		for i, t := range h.Library {
			if t.Name == tp.Name {
				newTp, err := h.TopicReader.Topic(t.Name)
				if err != nil {
					return nil
				}

				h.Library[i] = newTp
				break
			}
		}

	}

	return nil
}

func (t *Handler) completer() func(in prompt.Document) []prompt.Suggest {
	return func(in prompt.Document) []prompt.Suggest {

		s := []prompt.Suggest{}
		befCursor := in.TextBeforeCursor()

		// Only one character in line
		if "" == befCursor {
			return s
		}

		tokens := strings.Split(befCursor, " ")

		if len(tokens) == 1 {
			for _, tp := range t.Library {
				if strings.HasPrefix(tp.Name, befCursor) {
					s = append(s, prompt.Suggest{Text: tp.Name, Description: ""})
				}
			}

			return s
		}

		topicName := tokens[0]

		tp := topic.Topic{}
		for _, t := range t.Library {
			if t.Name == topicName {
				tp = t
				break
			}
		}

		// First token must be the topic
		if tp.Name == "" {
			return s
		}

		rest := strings.Join(tokens[1:], " ")

		if rest == "" {
			return s
		}

		for _, expr := range tp.Exprs {
			if strings.HasPrefix(expr.String(), rest) {
				// Do not show sugestion at the end of the text
				if len(rest) < len(expr.String()) {
					s = append(s, prompt.Suggest{Text: expr.String(), Description: ""})
				}
			}
		}

		return s
	}
}

func (h *Handler) parse(in string) (topic.Topic, topic.TopicExpr, int, error) {

	tp := topic.Topic{}

	tokens := strings.Fields(in)

	action := actionAdd
	if len(tokens) == 0 {
		return tp, nil, action, errors.New("No topic given to refine")
	}

	lastToken := tokens[len(tokens)-1]
	if strings.HasSuffix(lastToken, "/") {
		action = actionDelete
		tokens[len(tokens)-1] = lastToken[:len(lastToken)-1]
	}

	// First token must  be a valid topic
	for _, t := range h.Library {
		if strings.HasPrefix(t.Name, tokens[0]) {
			tp = t
			break
		}
	}

	if tp.Name == "" {
		return tp, nil, action, errors.New("There is no such topic: " + tokens[0] + ".")
	}

	expr := tokens[1:]
	if len(expr) == 0 {
		return tp, nil, action, errors.New("No expression given.")
	}

	exp, parseErr := topic.Parse(expr)
	if parseErr != nil {
		return tp, nil, action, parseErr
	}

	return tp, exp, action, nil
}

func exprExistInTopic(tp topic.Topic, expr topic.TopicExpr) bool {
	for _, e := range tp.Exprs {
		if topic.EqualExpr(e, expr) {
			return true
		}
	}

	return false
}

func removeExprFromTopic(tp topic.Topic, expr topic.TopicExpr) topic.Topic {

	exprs := make([]topic.TopicExpr, 0)

	for index, e := range tp.Exprs {
		if !topic.EqualExpr(e, expr) {
			continue
		}

		// Equal: append till index and after index
		exprs = append(exprs, tp.Exprs[:index]...)
		exprs = append(exprs, tp.Exprs[index+1:]...)
		break
	}

	return topic.Topic{Name: tp.Name, Exprs: exprs}
}
