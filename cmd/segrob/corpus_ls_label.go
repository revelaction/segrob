package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func corpusLsLabelCommand(repo storage.CorpusReader, opts CorpusLsLabelOptions, ui UI) error {
	var labels []string
	var err error

	if opts.ID != "" {
		// List labels for a specific document
		meta, err := repo.ReadMeta(opts.ID)
		if err != nil {
			return err
		}
		if meta.Labels != "" {
			// Split and filter in memory
			all := strings.Split(meta.Labels, ",")
			for _, l := range all {
				if opts.Match == "" || strings.Contains(l, opts.Match) {
					labels = append(labels, l)
				}
			}
		}
	} else {
		// List all labels in the corpus
		labels, err = repo.ListLabels(opts.Match)
		if err != nil {
			return err
		}
	}

	if len(labels) > 0 {
		fmt.Fprintln(ui.Out, strings.Join(labels, ", "))
	}

	return nil
}
