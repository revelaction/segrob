package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func liveLsCommand(repo storage.DocReader, opts LiveLsOptions, ui UI) error {
	docs, err := repo.List()
	if err != nil {
		return err
	}

	allLabels, err := repo.ListLabels("")
	if err != nil {
		return err
	}

	// Reverse the Name->ID map to an ID->Name map for printing lookups
	labelMap := allLabels.Reverse()

	// Print header
	fmt.Fprintf(ui.Out, liveLsFmt, "ID", "TITLE", "CREATOR", "TRANSLATOR", "DATE", "LANG")

	for _, doc := range docs {
		// Collect labels for the document
		var labelParts []string
		for _, id := range doc.LabelIDs {
			if name, ok := labelMap[id]; ok {
				labelParts = append(labelParts, name)
			}
		}

		// Filter
		if opts.Match != "" {
			matched := false
			for _, part := range labelParts {
				if strings.Contains(part, opts.Match) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Print tabular row
		fmt.Fprintf(ui.Out, liveLsFmt,
			doc.Id,
			truncate(extractLabelValue(labelParts, "title:"), 25),
			truncate(extractLabelValue(labelParts, "creator:"), 14),
			truncate(extractLabelValue(labelParts, "translator:"), 14),
			extractLabelValue(labelParts, "date:"),
			extractLabelValue(labelParts, "language:"),
		)
	}

	return nil
}
