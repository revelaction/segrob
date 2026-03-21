package main

import (
	"os"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/storage"
	"golang.org/x/term"
)

func corpusEditCommand(tr storage.TopicRepository, opts CorpusEditOptions, ui UI) error {
	fd := int(os.Stdin.Fd())
	if state, err := term.GetState(fd); err == nil {
		defer term.Restore(fd, state)
	}

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	hdl := edit.NewHandler(topicLib, tr, tr)
	return hdl.Run()
}
