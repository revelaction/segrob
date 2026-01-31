package main

import (
	"fmt"

	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

// Query command
func queryCommand(dr storage.DocRepository, tr storage.TopicRepository, opts QueryOptions, ui UI) error {

	if p, ok := dr.(storage.Preloader); ok {
		err := p.Preload(func(current, total int, name string) {
			fmt.Fprintf(ui.Err, "\r📖 Loading docs: %d/%d (%s)...%s", current, total, name, render.ClearLine)
		})
		fmt.Fprint(ui.Err, "\n")

		if err != nil {
			return err
		}
	}

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	// now present the REPL and prepare for topic in the REPL
	t := query.NewHandler(dr, topicLib, r)
	return t.Run()
}
