package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func corpusSetLabelCommand(repo storage.CorpusRepository, opts CorpusSetLabelOptions, ui UI) error {

	if opts.Delete {
		if err := repo.DeleteLabel(opts.DocID, opts.Labels...); err != nil {
			return fmt.Errorf("failed to delete labels: %w", err)
		}
		return nil
	}

	if err := repo.AddLabel(opts.DocID, opts.Labels...); err != nil {
		return fmt.Errorf("failed to add labels: %w", err)
	}

	return nil
}
