package main

import "strings"

// Column format for ls tabular output.
// FLAGS(5) ID(16) TITLE(25) CREATOR(14) TRANSLATOR(14) DATE(4) LANG
const corpusLsFmt = "%-5s  %-16s  %-25s  %-14s  %-14s  %-4s  %s\n"

// liveLsFmt omits the FLAGS column.
// ID(16) TITLE(25) CREATOR(14) TRANSLATOR(14) DATE(4) LANG
const liveLsFmt = "%-16s  %-25s  %-14s  %-14s  %-4s  %s\n"

// truncate returns s shortened to max runes, with trailing "…" if truncated.
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// extractLabelValue returns the value for a given prefix from a slice of label strings.
// If the label is missing, it returns "-".
func extractLabelValue(labels []string, prefix string) string {
	for _, part := range labels {
		if strings.HasPrefix(part, prefix) {
			return part[len(prefix):]
		}
	}
	return "-"
}
