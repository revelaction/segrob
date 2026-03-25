package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func liveInitCommand(mgr storage.SchemaManager, opts LiveInitOptions, ui UI) error {

	err := mgr.Create("doc_canonical.sql")
	if err != nil {
		return err
	}

	err = mgr.Create("doc_optimization.sql")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(ui.Err, "Database initialized at: %s\n", opts.DbPath)
	if err != nil {
		return err
	}
	return nil
}
