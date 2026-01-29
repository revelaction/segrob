package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func exportTopicCommand(opts ExportTopicOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return err
	}
	defer pool.Close()
	src := zombiezen.NewTopicStore(pool)

	dst := filesystem.NewTopicStore(opts.To)

	topics, err := src.ReadAll()
	if err != nil {
		return err
	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to export topic %s: %w", tp.Name, err)
		}
	}

	fmt.Fprintf(ui.Out, "Successfully exported %d topics from %s to %s\n", len(topics), opts.From, opts.To)
	return nil
}
