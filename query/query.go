package query

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/topic"

	"github.com/c-bata/go-prompt"
	"github.com/revelaction/segrob/storage"
)

const (
	completionThreshold = 2

	// topicPrefix is the Character in the prompt that prefixes the topic
	topicPrefix = "/"
)

type Handler struct {
	DocRepo      storage.DocReader
	TopicLibrary topic.Library
	Renderer     *render.Renderer
}

func NewHandler(dr storage.DocReader, tl topic.Library, r *render.Renderer) *Handler {
	return &Handler{
		DocRepo:      dr,
		TopicLibrary: tl,
		Renderer:     r,
	}
}

func (h *Handler) Run() error {

	fmt.Println("🔑 Ctrl+X: Toggle prefix, Ctrl+F: next Format, 🔧 quit")
	// Get all topics from the library directly
	topicNames := h.TopicLibrary.Names()

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

		// Fetch doc names for rendering
		docList, err := h.DocRepo.List("")
		if err != nil {
			fmt.Printf("Error listing docs: %v\n", err)
			continue
		}
		docNames := make(map[int]string)
		for _, d := range docList {
			docNames[d.Id] = d.Title
		}

		// Extract lemmas from all relevant expressions (OR logic) for indexed retrieval.
		// We only extract positive lemmas to find candidates in the database.
		// Fine-grained matching (including negative '!' lemmas) is performed
		// by the Matcher on the retrieved candidates.
		queries := tp.LemmaSets()
		lemmas := expr.Lemmas()
		if len(lemmas) > 0 {
			queries = append(queries, lemmas)
		}

		limit := 2000 // Limit candidates per expression to avoid hang

		var results []*match.SentenceMatch

		for _, lemmas := range queries {
			cursor := storage.Cursor(0)
			fetched := 0
			// doc := sent.Doc{Tokens: make([][]sent.Token, 1)} // No longer needed
			for {
				// Fetch batch
				newCursor, err := h.DocRepo.FindCandidates(lemmas, []string{}, cursor, 500, func(s sent.Sentence) error {
					fetched++
					h.Renderer.AddDocName(s.DocId, docNames[s.DocId])

					// Use MatchSentence directly to avoid "Tártaro" bug (overwrite due to missing SentenceId)
					sm := matcher.MatchSentence(s)
					if sm != nil {
						results = append(results, sm)
					}
					return nil
				})
				if err != nil {
					fmt.Printf("Error fetching candidates: %v\n", err)
					break
				}
				if cursor == newCursor {
					break // No more progress
				}

				if fetched >= limit {
					break
				}
				cursor = newCursor
			}
		}

		// Sort results to match previous behavior (Relevance > DocID > SentenceID)
		// We implement the sort here since we bypassed matcher.Sentences()
		sort.Slice(results, func(i, j int) bool {
			if results[i].NumExprs != results[j].NumExprs {
				return results[i].NumExprs > results[j].NumExprs
			}
			if results[i].Sentence.DocId != results[j].Sentence.DocId {
				return results[i].Sentence.DocId < results[j].Sentence.DocId
			}
			return results[i].Sentence.Id < results[j].Sentence.Id
		})

		h.Renderer.Match(results)
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
