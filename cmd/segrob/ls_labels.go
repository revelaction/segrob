package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func lsLabelsCommand(repo storage.DocRepository, opts LsLabelsOptions, ui UI) error {
	labels, err := repo.Labels(opts.Match)
	if err != nil {
		return err
	}

	if len(labels) > 0 {
		fmt.Fprintln(ui.Out, strings.Join(labels, ", "))
	}

	return nil
}
