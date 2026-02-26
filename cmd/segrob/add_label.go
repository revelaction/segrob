package main

import (
	"fmt"
)

func addLabelCommand(p *Pool, opts AddLabelOptions, ui UI) error {
	repo, err := NewDocRepository(p, opts.DocPath)
	if err != nil {
		return err
	}

	if err := repo.AddLabel(opts.DocID, opts.Labels...); err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	fmt.Fprintf(ui.Err, "Successfully added %d labels to doc ID %s\n", len(opts.Labels), opts.DocID)
	return nil
}
