package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func liveSetLabelCommand(repo storage.DocRepository, opts LiveSetLabelOptions, ui UI) error {

	if opts.Delete {
		if err := repo.RemoveLabel(opts.DocID, opts.Labels...); err != nil {
			return fmt.Errorf("failed to remove labels: %w", err)
		}
		fmt.Fprintf(ui.Err, "Successfully removed %d labels from doc ID %s\n", len(opts.Labels), opts.DocID)
		return nil
	}

	if err := repo.AddLabel(opts.DocID, opts.Labels...); err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully added %d labels to doc ID %s\n", len(opts.Labels), opts.DocID)
	return nil
}
