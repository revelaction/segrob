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

	for _, doc := range docs {
		labels, err := repo.Labels(doc.Id)
		if err != nil {
			return err
		}

		var labelNames []string
		matchFound := (opts.Match == "")
		for _, l := range labels {
			labelNames = append(labelNames, l.Name)
			if opts.Match != "" && strings.Contains(l.Name, opts.Match) {
				matchFound = true
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
