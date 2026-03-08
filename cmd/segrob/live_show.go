package main

import (
	"github.com/revelaction/segrob/storage"
)

func liveShowCommand(repo storage.DocRepository, opts ShowOptions, id string, ui UI) error {
	sentences, err := repo.Nlp(id)
	if err != nil {
		return err
	}

	renderDoc(sentences, opts, ui)
	return nil
}
