package main

import (
	"github.com/revelaction/segrob/edit"
)

func editCommand(opts EditOptions, isFile bool, ui UI) error {

	th, err := getTopicHandler(opts.TopicPath, isFile)
	if err != nil {
		return err
	}

	topicLib, err := th.All()
	if err != nil {
		return err
	}

	hdl := edit.NewHandler(topicLib, th, th)
	return hdl.Run()
}
