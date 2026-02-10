package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func lsDocCommand(repo storage.DocRepository, opts LsDocOptions, ui UI) error {
	docs, err := repo.List(opts.Match)
	if err != nil {
		return err
	}

	for _, doc := range docs {
		if len(doc.Labels) > 0 {
			fmt.Fprintf(ui.Out, "📖 %d %s 🔖 %s\n", doc.Id, doc.Title, strings.Join(doc.Labels, ", "))
		} else {
			fmt.Fprintf(ui.Out, "📖 %d %s\n", doc.Id, doc.Title)
		}
	}

	return nil
}
