package main

import (
	"fmt"

	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

// topicCommand prints the expressions of a topic
func topicCommand(tr storage.TopicRepository, opts TopicOptions, name string, ui UI) error {

	// No name provided (list all)
	if name == "" {
		topicLib, err := tr.ReadAll()
		if err != nil {
			return err
		}

		for topicId, name := range topicLib.Names() {
			fmt.Fprintf(ui.Out, "📖 %d %s \n", topicId, name)
		}

		return nil
	}

	tp, err := tr.Read(name)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.Topic(tp.Exprs)
	return nil
}
