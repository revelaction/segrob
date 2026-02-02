package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
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

	matcher := match.NewMatcher(topic.Topic{})
	matcher.AddTopicExpr(expr)
	err := matchDocs(dr, matcher, opts, ui)
	if err != nil {
		return err
	}

	return nil
}

func matchDocs(dr storage.DocRepository, matcher *match.Matcher, opts ExprOptions, ui UI) error {

	if opts.Sent != nil {
		if opts.Doc == nil {
			return errors.New("--sent flag given but no --doc")
		}
	}

	r := render.NewRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	if opts.Doc != nil {
		docId := *opts.Doc
		doc, err := dr.Read(docId)
		if err != nil {
			return err
		}

		doc.Id = docId

		if opts.Sent != nil {
			doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*opts.Sent]}}
		}

		matcher.Match(doc)

	} else {
		// Optimized search using FindCandidates for SQLite performance.
		// We extract positive lemmas to narrow down candidate sentences using the index.
		// The Matcher then performs the full expression evaluation on these candidates.
		lemmas := matcher.ArgExpr.Lemmas()
		if len(lemmas) == 0 {
			return errors.New("expression must contain at least one lemma for indexing")
		}

		// Fetch doc IDs and titles for metadata mapping
		docs, err := dr.List()
		if err != nil {
			return err
		}
		docNames := make(map[int]string)
		for _, d := range docs {
			docNames[d.Id] = d.Title
			r.AddDocName(d.Id, d.Title)
		}

		cursor := storage.Cursor(0)
		for {
			newCursor, err := dr.FindCandidates(lemmas, cursor, 1000, func(res storage.SentenceResult) error {
				// Construct a valid doc with single sentence for matching
				doc := sent.Doc{
					Id:     res.DocID,
					Title:  docNames[res.DocID],
					Tokens: [][]sent.Token{res.Tokens},
				}
				matcher.Match(doc)
				return nil
			})
			if err != nil {
				return err
			}
			if cursor == newCursor {
				break // No more candidates
			}
			cursor = newCursor
		}
	}

	result := matcher.Sentences()

	r.Match(result)
	return nil
}

func hasLabels(fileLabels, cmdLabels []string) bool {
	// No command line labels to match
	if nil == cmdLabels {
		return true
	}

	for _, label := range cmdLabels {

		isLabel := false
		for _, l := range fileLabels {
			if strings.Contains(l, label) {
				isLabel = true
			}
		}

		if !isLabel {
			return false
		}
	}

	return true
}
