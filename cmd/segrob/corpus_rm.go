package main

import (
	"github.com/revelaction/segrob/storage"
)

func corpusRmCommand(repo storage.CorpusRepository, opts CorpusRmOptions, ui UI) error {
	// 1. Remove the document
	if err := repo.Delete(opts.ID); err != nil {
		return err
	}

	return nil
}
