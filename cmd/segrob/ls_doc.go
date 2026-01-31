package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func lsDocCommand(repo storage.DocRepository, opts LsDocOptions, ui UI) error {
	docs, err := repo.List()
	if err != nil {
		return err
	}

	for _, doc := range docs {
		fmt.Fprintf(ui.Out, "📖 %d %s\n", doc.Id, doc.Title)
	}

	return nil
}
