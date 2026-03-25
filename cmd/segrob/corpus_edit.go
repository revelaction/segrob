package main

import (
	"errors"
	"os"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/storage"
	"golang.org/x/term"
)

func corpusEditCommand(tr storage.TopicRepository, opts CorpusEditOptions, ui UI) (err error) {
	fd := int(os.Stdin.Fd())
	state, gErr := term.GetState(fd)
	if gErr == nil {
		defer func() {
			err = errors.Join(err, term.Restore(fd, state))
		}()
	}

	topicLib, rErr := tr.ReadAll()
	if rErr != nil {
		return rErr
	}

	hdl := edit.NewHandler(topicLib, tr, tr)
	hdlErr := hdl.Run()
	if hdlErr != nil {
		return hdlErr
	}

	return nil
}
