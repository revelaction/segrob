package main

import (
	"fmt"
	"github.com/revelaction/segrob/storage"
)

func corpusRmCommand(repo storage.CorpusRepository, opts CorpusRmOptions, ui UI) error {
	// 1. Remove the document
	if err := repo.Delete(opts.ID); err != nil {
		return err
	}

	// 2. Report success to stderr (ui.Err)
	fmt.Fprintf(ui.Err, "Document %s removed successfully.\n", opts.ID)
	return nil
}
