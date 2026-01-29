package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
)

// topicCommand prints the expressions of a topic
func topicCommand(opts TopicOptions, name string, isFile bool, ui UI) error {

	fhr, err := getTopicHandler(opts.TopicPath, isFile)
	if err != nil {
		return err
	}

	// No name provided (list all)
	if name == "" {
		topicLib, err := fhr.ReadAll()
		if err != nil {
			return err
		}

		for topicId, name := range topicLib.Names() {
			fmt.Fprintf(ui.Out, "📖 %d %s \n", topicId, name)
		}

		return nil
	}

	tp, err := fhr.Read(name)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.Topic(tp.Exprs)
	return nil
}
