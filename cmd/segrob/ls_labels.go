package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func lsLabelsCommand(repo storage.DocRepository, opts LsLabelsOptions, ui UI) error {
	labels, err := repo.ListLabels(opts.Match)
	if err != nil {
		return err
	}

	if len(labels) > 0 {
		var names []string
		for _, l := range labels {
			names = append(names, l.Name)
		}
		fmt.Fprintln(ui.Out, strings.Join(names, ", "))
	}

	return nil
}
