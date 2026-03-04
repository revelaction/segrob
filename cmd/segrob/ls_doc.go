package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func lsDocCommand(repo storage.DocRepository, opts LsDocOptions, ui UI) error {
	docs, err := repo.List()
	if err != nil {
		return err
	}

	allLabels, err := repo.ListLabels("")
	if err != nil {
		return err
	}

	labelMap := make(map[int]string)
	for _, l := range allLabels {
		labelMap[l.ID] = l.Name
	}

	for _, doc := range docs {
		var labelNames []string
		matchFound := (opts.Match == "")
		for _, id := range doc.LabelIDs {
			if name, ok := labelMap[id]; ok {
				labelNames = append(labelNames, name)
				if opts.Match != "" && strings.Contains(name, opts.Match) {
					matchFound = true
				}
			}
		}

		if !matchFound {
			continue
		}

		if len(labelNames) > 0 {
			fmt.Fprintf(ui.Out, "📖 %s %s 🔖 %s\n", doc.Id, doc.Source, strings.Join(labelNames, ", "))
		} else {
			fmt.Fprintf(ui.Out, "📖 %s %s\n", doc.Id, doc.Source)
		}
	}

	return nil
}
