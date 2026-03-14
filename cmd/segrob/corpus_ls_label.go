package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func corpusLsLabelCommand(repo storage.CorpusReader, opts CorpusLsLabelOptions, ui UI) error {
	labels, err := repo.ListLabels(opts.Match)
	if err != nil {
		return err
	}

	if len(labels) > 0 {
		fmt.Fprintln(ui.Out, strings.Join(labels, ", "))
	}

	return nil
}
