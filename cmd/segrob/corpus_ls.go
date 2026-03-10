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
		if opts.NlpAck && !r.NlpAck {
			continue
		}
		if opts.TxtAck && !r.TxtAck {
			continue
		}
		if opts.Ack && !r.HasAck() {
			continue
		}

		txtStatus := "❌"
		if r.HasTxt() {
			txtStatus = "✅"
			if r.TxtAck {
				txtStatus = "👍"
			}
		}
		nlpStatus := "❌"
		if r.HasNlp() {
			nlpStatus = "✅"
			if r.NlpAck {
				nlpStatus = "👍"
			}
		}
		ackStatus := "❌"
		if r.HasAck() {
			ackStatus = "👍"
		}
		fmt.Fprintf(ui.Out, "📖 %s 🔖 %s txt:%s nlp:%s ack:%s\n", r.ID, r.Labels, txtStatus, nlpStatus, ackStatus)
	}

	return nil
}
