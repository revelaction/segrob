package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func initDbCommand(p *Pool, opts InitDbOptions, ui UI) error {
	pool, err := p.Open(opts.DbPath)
	if err != nil {
		return err
	}

	if err := zombiezen.CreateSchemas(pool, "doc_canonical.sql"); err != nil {
		return err
	}

	if err := zombiezen.CreateSchemas(pool, "doc_optimization.sql"); err != nil {
		return err
	}

	fmt.Fprintf(ui.Out, "Database initialized at: %s\n", opts.DbPath)
	return nil
}
