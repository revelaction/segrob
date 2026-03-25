package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func corpusInitCommand(mgr storage.SchemaManager, opts CorpusInitOptions, ui UI) error {

	if err := mgr.Create("corpus.sql"); err != nil {
		return err
	}

	_, err := fmt.Fprintf(ui.Err, "Corpus database initialized at: %s\n", opts.DbPath)
	return err
}
