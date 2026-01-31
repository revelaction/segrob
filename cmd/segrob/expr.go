package main

import (
	"errors"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
)

func exprCommand(dr storage.DocRepository, opts ExprOptions, args []string, ui UI) error {

	// args is guaranteed to have at least 1 element by parseExprArgs
	// parse the expr expression
	expr, parseErr := topic.Parse(args)
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
		docs, err := dr.List()
		if err != nil {
			return err
		}

		for _, metaDoc := range docs {
			if !hasLabels(metaDoc.Labels, opts.Labels) {
				continue
			}

			doc, err := dr.Read(metaDoc.Id)
			if err != nil {
				return err
			}

			doc.Title = metaDoc.Title
			doc.Labels = metaDoc.Labels
			r.AddDocName(metaDoc.Id, doc.Title)
			matcher.Match(doc)
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
