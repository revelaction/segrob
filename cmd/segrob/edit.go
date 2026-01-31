package main

import (
	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/storage"
)

func editCommand(tr storage.TopicRepository, opts EditOptions, ui UI) error {

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	hdl := edit.NewHandler(topicLib, tr, tr)
	return hdl.Run()
}
