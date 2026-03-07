package main

import (
	"github.com/revelaction/segrob/storage"
)

func docCommand(repo storage.DocRepository, opts DocOptions, id string, ui UI) error {
	sentences, err := repo.Nlp(id)
	if err != nil {
		return err
	}

	renderDoc(sentences, opts, ui)
	return nil
}
