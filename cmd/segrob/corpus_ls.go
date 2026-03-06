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
		if opts.WithNlp && !r.HasNlp() {
			continue
		}
		txtStatus := "❌"
		if r.HasTxt() {
			txtStatus = "✅"
		}
		nlpStatus := "❌"
		if r.HasNlp() {
			nlpStatus = "✅"
		}
		fmt.Fprintf(ui.Out, "📖 %s 🔖 %s txt: %s nlp: %s\n", r.ID, r.Labels, txtStatus, nlpStatus)
	}

	return nil
}
