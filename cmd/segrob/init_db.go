package main

import (
	"fmt"

	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func liveInitCommand(pool *sqlitex.Pool, opts LiveInitOptions, ui UI) error {

	if err := zombiezen.CreateSchemas(pool, "doc_canonical.sql"); err != nil {
		return err
	}

	if err := zombiezen.CreateSchemas(pool, "doc_optimization.sql"); err != nil {
		return err
	}

	fmt.Fprintf(ui.Out, "Database initialized at: %s\n", opts.DbPath)
	return nil
}
