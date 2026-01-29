package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func importTopicCommand(opts ImportTopicOptions, ui UI) error {

	src := filesystem.NewTopicStore(opts.From)

	pool, err := zombiezen.NewPool(opts.To)

	if err != nil {

		return err

	}

	defer pool.Close()

	if err := zombiezen.CreateTopicTables(pool); err != nil {

		return fmt.Errorf("failed to create topics table: %w", err)

	}

	dst := zombiezen.NewTopicStore(pool)

	topics, err := src.List()

	if err != nil {

		return err

	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to import topic %s: %w", tp.Name, err)
		}
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully imported %d topics from %s to %s\n", len(topics), opts.From, opts.To)
	return nil
}
