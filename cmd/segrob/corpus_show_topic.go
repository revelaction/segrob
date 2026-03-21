package main

import (
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
)

func corpusShowTopicCommand(tr storage.TopicRepository, opts CorpusShowTopicOptions, name string, ui UI) error {
	tp, err := tr.Read(name)
	if err != nil {
		return err
	}

	r := render.NewCLIRenderer()
	r.Topic(tp.Exprs)
	return nil
}
