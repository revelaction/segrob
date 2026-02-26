package main

import (
	"fmt"
)

func removeLabelCommand(p *Pool, opts RemoveLabelOptions, ui UI) error {
	repo, err := NewDocRepository(p, opts.DocPath)
	if err != nil {
		return err
	}

	if err := repo.RemoveLabel(opts.DocID, opts.Labels...); err != nil {
		return fmt.Errorf("failed to remove labels: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully removed %d labels from doc ID %s\n", len(opts.Labels), opts.DocID)
	return nil
}
