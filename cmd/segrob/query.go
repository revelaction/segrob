package main

import (
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

// Query command
func queryCommand(dr storage.DocRepository, tr storage.TopicRepository, opts QueryOptions, ui UI) error {

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	r := render.NewCLIRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	// now present the REPL and prepare for topic in the REPL
	t := query.NewHandler(dr, topicLib, r, opts.Labels)
	return t.Run()
}
