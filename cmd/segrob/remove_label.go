package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func removeLabelCommand(repo storage.DocRepository, opts RemoveLabelOptions, ui UI) error {

	if err := repo.RemoveLabel(opts.DocID, opts.Labels...); err != nil {
		return fmt.Errorf("failed to remove labels: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully removed %d labels from doc ID %s\n", len(opts.Labels), opts.DocID)
	return nil
}
