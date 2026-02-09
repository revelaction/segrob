package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/search"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
)

func exprCommand(dr storage.DocRepository, opts ExprOptions, args []string, ui UI) error {

	if p, ok := dr.(storage.Preloader); ok {
		err := p.Preload(func(current, total int, name string) {
			fmt.Fprintf(ui.Err, "\r📖 Loading docs: %d/%d (%s)...%s", current, total, name, render.ClearLine)
		})
		fmt.Fprint(ui.Err, "\n")

		if err != nil {
			return err
		}
	}

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

	// Strategy Selection
	s := search.New(topic.Topic{}, dr)
	if opts.Doc != nil {
		s.WithDocID(*opts.Doc)
	}
	if len(opts.Labels) > 0 {
		s.WithLabels(opts.Labels)
	}

	// Prepare results accumulator
	var results []*match.SentenceMatch
	onMatch := func(m *match.SentenceMatch) error {
		results = append(results, m)
		return nil
	}

	// Execute search with pagination
	cursor := storage.Cursor(0)
	limit := 1000
	for {
		newCursor, err := s.Sentences(expr, cursor, limit, onMatch)
		if err != nil {
			return err
		}
		if cursor == newCursor {
			break
		}
		cursor = newCursor
	}

	// Render results
	r := render.NewRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	if opts.Doc == nil {
		// Populate DocNames for indexed search
		list, err := dr.List()
		if err != nil {
			return err
		}
		for _, d := range list {
			r.AddDocName(d.Id, d.Title)
		}
	}

	r.Match(results)

	return nil
}
