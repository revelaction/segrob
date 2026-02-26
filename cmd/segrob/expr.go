package main

import (
	"os"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
)

func exprCommand(dr storage.DocRepository, opts ExprOptions, args []string, ui UI) error {

	// args is guaranteed to have at least 1 element by parseExprArgs
	// Flatten arguments to support quoted expressions containing spaces,
	// matching the behavior of the query REPL.
	//
	// Verification use cases:
	// - Unquoted: segrob expr a 1 el      -> args:["a", "1", "el"] -> flatArgs:["a", "1", "el"]
	// - Quoted:   segrob expr "a 1 el"    -> args:["a 1 el"]       -> flatArgs:["a", "1", "el"]
	// - Mixed:    segrob expr "a 1" el    -> args:["a 1", "el"]    -> flatArgs:["a", "1", "el"]
	var flatArgs []string
	for _, arg := range args {
		flatArgs = append(flatArgs, strings.Fields(arg)...)
	}

	// parse the expr expression
	expr, parseErr := topic.Parse(flatArgs)
	if parseErr != nil {
		return parseErr
	}

	// Resolve labels to IDs
	var labelIDs []int
	if len(opts.Labels) > 0 {
		allLabels, err := dr.ListLabels("")
		if err != nil {
			return err
		}
		labelMap := make(map[string]int)
		for _, l := range allLabels {
			labelMap[l.Name] = l.ID
		}
		for _, name := range opts.Labels {
			if id, ok := labelMap[name]; ok {
				labelIDs = append(labelIDs, id)
			}
		}
	}

	// Prepare Matcher
	matcher := match.NewMatcher(topic.Topic{})
	matcher.AddTopicExpr(expr)

	// Prepare results accumulator
	var results []*match.SentenceMatch
	limitReached := false
	onMatch := func(m *match.SentenceMatch) error {
		results = append(results, m)
		if opts.Limit > 0 && len(results) >= opts.Limit {
			limitReached = true
		}
		return nil
	}

	// Execute search with pagination
	cursor := storage.Cursor(0)
	limit := 1000
	lemmas := expr.Lemmas()

	for {
		newCursor, err := dr.FindCandidates(lemmas, labelIDs, cursor, limit, func(s sent.Sentence) error {
			if m := matcher.MatchSentence(s); m != nil {
				return onMatch(m)
			}
			return nil
		})

		if err != nil {
			return err
		}
		if cursor == newCursor {
			break
		}
		if limitReached {
			break
		}
		cursor = newCursor
	}

	// Render results
	if opts.JSON {
		jr := render.NewJSONRenderer(os.Stdout)
		jr.Render(results)
		return nil
	}

	r := render.NewCLIRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	// Populate DocNames for indexed search
	list, err := dr.List()
	if err != nil {
		return err
	}
	for _, d := range list {
		r.AddDocName(d.Id, d.Source)
	}

	r.Render(results)

	return nil
}
