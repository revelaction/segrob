package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func liveLsLabelCommand(repo storage.DocReader, opts LiveLsLabelOptions, ui UI) error {

	labels, err := repo.ListLabels(opts.Match)
	if err != nil {
		return err
	}

	if len(labels) > 0 {
		// Use Go 1.23 standard library packages to extract and sort keys
		names := slices.Sorted(maps.Keys(labels))
		fmt.Fprintln(ui.Out, strings.Join(names, ", "))
	}

	return nil
}
