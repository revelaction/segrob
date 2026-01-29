package main

import (
	"github.com/gosuri/uiprogress"
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

// Query command
func queryCommand(opts QueryOptions, isTopicFile, isDocFile bool, ui UI) error {

	var dr storage.DocRepository
	path := opts.DocPath

	if isDocFile {
		pool, err := zombiezen.NewPool(path)
		if err != nil {
			return err
		}
		dr = zombiezen.NewDocStore(pool)
	} else {
		h, err := filesystem.NewDocStore(path)
		if err != nil {
			return err
		}

		if err := h.LoadList(); err != nil {
			return err
		}

		uiprogress.Start()
		bar := uiprogress.AddBar(1) // Placeholder, updated in callback
		bar.AppendCompleted()
		bar.PrependElapsed()

		var currentName string
		bar.AppendFunc(func(b *uiprogress.Bar) string {
			return currentName
		})

		err = h.LoadContents(func(total int, name string) {
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
		dr = h
	}

	th, err := getTopicHandler(opts.TopicPath, isTopicFile)
	if err != nil {
		return err
	}
	topicLib, err := th.ReadAll()
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
