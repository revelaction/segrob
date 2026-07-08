package main

import (
	"fmt"
	"os"
	"strings"
)

func envCommand(ui UI) error {
	vars := []struct {
		name string
	}{
		{"SEGROB_CORPUS_DB"},
		{"SEGROB_LIVE_DB"},
		{"SEGROB_NLP_SCRIPT"},
	}

	var b strings.Builder
	for _, v := range vars {
		val := os.Getenv(v.name)
		if val == "" {
			val = "(not set)"
		}
		_, _ = fmt.Fprintf(&b, "%s=%s\n", v.name, val)
	}

	_, err := fmt.Fprint(ui.Out, b.String())
	return err
}
