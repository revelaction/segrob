package main

import (
	"fmt"
	"strings"

	"github.com/revelaction/segrob/storage"
)

// Column format for corpus ls tabular output.
// FLAGS(5) ID(16) TITLE(25) CREATOR(14) TRANSLATOR(14) DATE(4) LANG
const corpusLsFmt = "%-5s  %-16s  %-25s  %-14s  %-14s  %-4s  %s\n"

func corpusLsCommand(repo storage.CorpusReader, opts CorpusLsOptions, ui UI) error {
	metas, err := repo.List()
	if err != nil {
		return err
	}

	var matches []storage.CorpusMeta
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
		matches = append(matches, m)
	}

	if len(matches) == 0 {
		return nil
	}

	// Print header
	fmt.Fprintf(ui.Out, corpusLsFmt, "FLAGS", "ID", "TITLE", "CREATOR", "TRANSLATOR", "DATE", "LANG")

	for _, m := range matches {
		fmt.Fprintf(ui.Out, corpusLsFmt,
			corpusChars(m),
			m.ID,
			truncate(extractLabelValue(m.Labels, "title:"), 25),
			truncate(extractLabelValue(m.Labels, "creator:"), 14),
			truncate(extractLabelValue(m.Labels, "translator:"), 14),
			extractLabelValue(m.Labels, "date:"),
			extractLabelValue(m.Labels, "language:"),
		)
	}

	return nil
}

// truncate returns s shortened to max runes, with trailing "…" if truncated.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// extractLabelValue returns the value for a given prefix from the comma-separated Labels string.
// If the label is missing, it returns "-".
// Example: extractLabelValue("creator:carroll,title:foo", "creator:") → "carroll"
func extractLabelValue(labels, prefix string) string {
	for _, part := range strings.Split(labels, ",") {
		if strings.HasPrefix(part, prefix) {
			return part[len(prefix):]
		}
	}
	return "-"
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
