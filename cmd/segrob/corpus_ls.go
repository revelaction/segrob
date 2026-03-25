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

	var matches []storage.CorpusMeta
	filter := storage.NormalizeLabel(opts.Filter)

	for _, m := range metas {
		if !strings.Contains(m.Labels, filter) {
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
		matches = append(matches, m)
	}

	if len(matches) == 0 {
		return nil
	}

	// Print header
	_, err = fmt.Fprintf(ui.Out, corpusLsFmt, "FLAGS", "ID", "TITLE", "CREATOR", "TRANSLATOR", "DATE", "LANG")
	if err != nil {
		return err
	}

	for _, m := range matches {
		labelParts := strings.Split(m.Labels, ",")

		_, err = fmt.Fprintf(ui.Out, corpusLsFmt,
			corpusChars(m),
			m.ID,
			truncate(extractLabelValue(labelParts, "title:"), 25),
			truncate(extractLabelValue(labelParts, "creator:"), 14),
			truncate(extractLabelValue(labelParts, "translator:"), 14),
			extractLabelValue(labelParts, "date:"),
			extractLabelValue(labelParts, "language:"),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// corpusChars returns a 5-char status string for the given CorpusMeta.
func corpusChars(m storage.CorpusMeta) string {
	ch := func(set bool, c byte) byte {
		if set {
			return c
		}
		return '-'
	}
	return string([]byte{
		ch(m.HasTxt(), 't'),
		ch(m.TxtEdit, 'e'),
		ch(m.TxtAck, 'a'),
		ch(m.HasNlp(), 'n'),
		ch(m.NlpAck, 'a'),
	})
}
