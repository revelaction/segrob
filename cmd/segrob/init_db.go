package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func liveInitCommand(mgr storage.SchemaManager, opts LiveInitOptions, ui UI) error {

	if err := mgr.Create("doc_canonical.sql"); err != nil {
		return err
	}

	if err := mgr.Create("doc_optimization.sql"); err != nil {
		return err
	}

	fmt.Fprintf(ui.Err, "Database initialized at: %s\n", opts.DbPath)
	return nil
}
