package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func corpusAckCommand(repo storage.CorpusRepository, opts CorpusAckOptions, ui UI) error {
	var err error
	if opts.Nlp {
		err = repo.AckNlp(opts.ID, opts.By)
	} else {
		err = repo.AckTxt(opts.ID, opts.By)
	}

	if err != nil {
		return fmt.Errorf("failed to acknowledge for %s: %w", opts.ID, err)
	}

	target := "txt"
	if opts.Nlp {
		target = "nlp"
	}
	fmt.Fprintf(ui.Err, "Successfully acknowledged %s for %s\n", target, opts.ID)
	return nil
}
