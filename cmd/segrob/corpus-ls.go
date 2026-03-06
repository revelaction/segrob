package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func corpusLsCommand(repo storage.CorpusReader, opts CorpusLsOptions, ui UI) error {
	records, err := repo.List()
	if err != nil {
		return err
	}

	for _, r := range records {
		if opts.Filter != "" && !strings.Contains(r.Labels, opts.Filter) {
			continue
		}
		status := "❌"
		if r.HasTxt() {
			status = "✅"
		}
		fmt.Fprintf(ui.Out, "📖 %s 🔖 %s txt: %s\n", r.ID, r.Labels, status)
	}

	return nil
}
