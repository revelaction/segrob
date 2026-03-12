package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

func corpusLsCommand(repo storage.CorpusReader, opts CorpusLsOptions, ui UI) error {
	metas, err := repo.List()
	if err != nil {
		return err
	}

	for _, m := range metas {
		if opts.Filter != "" && !strings.Contains(m.Labels, opts.Filter) {
			continue
		}
		if opts.WithNlp && !m.HasNlp() {
			continue
		}
		if opts.NlpAck && !m.NlpAck {
			continue
		}
		if opts.TxtAck && !m.TxtAck {
			continue
		}
		if opts.Ack && !m.HasAck() {
			continue
		}

		txtStatus := "❌"
		if m.HasTxt() {
			txtStatus = "✅"
			if m.TxtAck {
				txtStatus = "👍"
			}
		}
		nlpStatus := "❌"
		if m.HasNlp() {
			nlpStatus = "✅"
			if m.NlpAck {
				nlpStatus = "👍"
			}
		}
		ackStatus := "❌"
		if m.HasAck() {
			ackStatus = "👍"
		}
		fmt.Fprintf(ui.Out, "📖 %s 🔖 %s txt:%s nlp:%s ack:%s\n", m.ID, m.Labels, txtStatus, nlpStatus, ackStatus)
	}

	return nil
}
