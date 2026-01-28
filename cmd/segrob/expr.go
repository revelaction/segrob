package main

import (
	"errors"
	"strings"

	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"github.com/revelaction/segrob/topic"
)

func exprCommand(opts ExprOptions, args []string, isDocFile bool, ui UI) error {

	// args is guaranteed to have at least 1 element by parseExprArgs
	// parse the expr expression
	expr, parseErr := topic.Parse(args)
	if parseErr != nil {
		return parseErr
	}

	matcher := match.NewMatcher(topic.Topic{})
	matcher.AddTopicExpr(expr)
	err := matchDocs(matcher, opts, isDocFile, ui)
	if err != nil {
		return err
	}

	return nil
}

func matchDocs(matcher *match.Matcher, opts ExprOptions, isDocFile bool, ui UI) error {

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

	var dr storage.DocRepository
	path := opts.DocPath

	if isDocFile {
		pool, err := zombiezen.NewPool(path)
		if err != nil {
			return err
		}
		dr = zombiezen.NewDocHandler(pool)
	} else {
		h, err := filesystem.NewDocHandler(path)
		if err != nil {
			return err
		}
		if err := h.Load(nil); err != nil {
			return err
		}
		dr = h
	}

	if opts.Doc != nil {
		docId := *opts.Doc
		doc, err := dr.Doc(docId)
		if err != nil {
			return err
		}

		doc.Id = docId

		if opts.Sent != nil {
			doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*opts.Sent]}}
		}

		matcher.Match(doc)

	} else {
		docNames, err := dr.Names()
		if err != nil {
			return err
		}

		for docId, name := range docNames {

			doc, err := dr.DocForName(name)
			if err != nil {
				return err
			}

			if !hasLabels(doc.Labels, opts.Labels) {
				continue
			}

			doc.Id = docId
			r.AddDocName(docId, doc.Title)
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
