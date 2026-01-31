package main

import (
	"github.com/gosuri/uiprogress"
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

// Query command
func queryCommand(dr storage.DocRepository, tr storage.TopicRepository, opts QueryOptions, ui UI) error {

	if p, ok := dr.(storage.Preloader); ok {
		uiprogress.Start()
		bar := uiprogress.AddBar(1) // Placeholder, updated in callback
		bar.AppendCompleted()
		bar.PrependElapsed()

		var currentName string
		bar.AppendFunc(func(b *uiprogress.Bar) string {
			return currentName
		})

		err := p.Preload(func(total int, name string) {
			if bar.Total <= 1 {
				bar.Total = total
				bar.Set(0)
			}
			currentName = name
			bar.Incr()
		})
		uiprogress.Stop()

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
