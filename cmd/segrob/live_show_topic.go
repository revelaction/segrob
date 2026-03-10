package main

import (
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

// liveShowTopicCommand prints the expressions of a topic
func liveShowTopicCommand(tr storage.TopicRepository, opts LiveShowTopicOptions, name string, ui UI) error {

	tp, err := tr.Read(name)
	if err != nil {
		return err
	}

	r := render.NewCLIRenderer()
	r.Topic(tp.Exprs)
	return nil
}
