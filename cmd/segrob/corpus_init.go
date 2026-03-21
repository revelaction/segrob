package main

import (
	"fmt"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func corpusInitCommand(pool *sqlitex.Pool, opts CorpusInitOptions, ui UI) error {

	if err := zombiezen.CreateSchemas(pool, "corpus.sql"); err != nil {
		return err
	}

	fmt.Fprintf(ui.Err, "Corpus database initialized at: %s\n", opts.DbPath)
	return nil
}
