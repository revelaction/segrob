package main

import (
	"fmt"
)

func versionCommand(ui UI) error {
	_, err := fmt.Fprintf(ui.Out, "segrob version %s (commit: %s)\n", BuildTag, BuildCommit)
	return err
}
