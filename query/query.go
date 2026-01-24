package query

import (
	"errors"
	"fmt"
	"github.com/revelaction/segrob/file"
	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"
	"strings"

	prompt "github.com/c-bata/go-prompt"
)

const (
	completionThreshold = 2

	// topicPrefix is the Character in the prompt that prefixes the topic
	topicPrefix = "/"
)

type Handler struct {
	Library      sent.Library
	TopicLibrary topic.Library
	Renderer     *render.Renderer
}

func NewHandler(l sent.Library, tl topic.Library, r *render.Renderer) *Handler {
	return &Handler{
		Library:      l,
		TopicLibrary: tl,
		Renderer:     r,
	}
}

func (h *Handler) Run() error {

	fmt.Println("🔑 Ctrl+X: Toggle prefix, Ctrl+F: next Format, 🔧 quit")
	// Get all topics
	th := file.NewTopicHandler(file.TopicDir)

	topicNames, err := th.Names()
	if err != nil {
		return err
	}

	// initialize prompt history
	history := []string{}

	for {

		in := prompt.Input("      🔖 ", h.completer(topicNames),
			prompt.OptionTitle("segrob query"),
			prompt.OptionPrefixTextColor(prompt.Yellow),
			prompt.OptionPreviewSuggestionTextColor(prompt.Blue),
			prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
			prompt.OptionMaxSuggestion(12),
			prompt.OptionSuggestionBGColor(prompt.DarkGray),
			prompt.OptionHistory(history),
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.ControlF,
				Fn: func(buf *prompt.Buffer) {
					h.Renderer.NextFormat()
					fmt.Println("Format set to: " + h.Renderer.Format)
				}}),
			prompt.OptionAddKeyBind(prompt.KeyBind{
				Key: prompt.ControlX,
				Fn: func(buf *prompt.Buffer) {
					h.Renderer.NextPrefix()
					fmt.Println("Prefix set to " + fmt.Sprintf("%t", h.Renderer.HasPrefix))
				}}),
		)

		if in == "quit" {
			return nil
		}

		history = append(history, in)
		// return the topic
		tp, expr, err := h.parse(in)
		if err != nil {
			continue
		}

		matcher := match.NewMatcher(tp)
		matcher.AddTopicExpr(expr)

		for _, doc := range h.Library {
			h.Renderer.AddDocName(doc.Id, doc.Title)
			matcher.Match(doc)
		}

		result := matcher.Sentences()
		h.Renderer.Match(result)
	}

	return nil
}

func (h *Handler) completer(topicNames []string) func(in prompt.Document) []prompt.Suggest {
	return func(in prompt.Document) []prompt.Suggest {

		s := []prompt.Suggest{}
		befCursor := in.TextBeforeCursor()

		// Only one character in line
		if "" == befCursor {
			return s
		}

		tokens := strings.Split(befCursor, " ")
		firstToken := tokens[0]

		if len(tokens) == 1 {
			s = append(s, h.completeTopic(firstToken)...)
			s = append(s, h.completeExpressionItem(firstToken)...)
			return s
		}

		// len > 1
		isFirstTopic := false
		for _, t := range h.TopicLibrary {
			if t.Name == firstToken {
				isFirstTopic = true
				break
			}
		}

		// len = 2 and first is topic
		if len(tokens) == 2 {
			if isFirstTopic {
				s = append(s, h.completeExpressionItem(tokens[1])...)
			}

			return s
		}

		// len > 2, complete as expr string
		rest := befCursor

		if isFirstTopic {
			rest = befCursor[len(firstToken)+1:]
		}

		for _, topic := range h.TopicLibrary {
			for _, expr := range topic.Exprs {
				if len(rest) > len(expr.String()) {
					continue
				}

				//
				if strings.HasPrefix(expr.String(), rest) {
					wordBeforeLen := len(in.GetWordBeforeCursor())

					start := len(rest) - wordBeforeLen
					restExpr := expr.String()[start:]
					s = append(s, prompt.Suggest{Text: restExpr, Description: topic.Name})
					continue
				}
			}
		}

		return s
	}
}

func (h *Handler) completeTopic(token string) (s []prompt.Suggest) {
	for _, tp := range h.TopicLibrary {
		if strings.HasPrefix(tp.Name, token) {
			s = append(s, prompt.Suggest{Text: tp.Name, Description: "🔖 " + tp.Name})
		}
	}

	return s
}

func (h *Handler) completeExpressionItem(token string) (s []prompt.Suggest) {
	for _, topic := range h.TopicLibrary {
		for _, expr := range topic.Exprs {
			for _, exprItem := range expr {
				// Lemma
				if strings.HasPrefix(exprItem.Lemma, token) {
					s = append(s, prompt.Suggest{Text: expr.String(), Description: topic.Name})
					continue
				}

				// Tag
				if strings.HasPrefix(exprItem.Tag, token) {
					s = append(s, prompt.Suggest{Text: expr.String(), Description: topic.Name})
				}
			}
		}
	}

	return s
}

func (h *Handler) parse(in string) (topic.Topic, topic.TopicExpr, error) {

	tp := topic.Topic{}

	tokens := strings.Fields(in)

	if len(tokens) == 0 {
		return tp, nil, errors.New("No topic given to refine")
	}

	isFirstTopic := false

	// First token must  be a valid topic
	for _, t := range h.TopicLibrary {
		if t.Name == tokens[0] {
			isFirstTopic = true
			tp = t
			break
		}
	}

	expr := tokens
	if isFirstTopic {
		expr = tokens[1:]
	}

	if len(expr) == 0 {
		if !isFirstTopic {
			return tp, nil, errors.New("There are no topic and no expr")
		}
	}

	exp, parseErr := topic.Parse(expr)
	if parseErr != nil {
		return tp, nil, parseErr
	}

	return tp, exp, nil
}
