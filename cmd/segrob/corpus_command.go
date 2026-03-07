package main

import (
	"fmt"
	"io"
	"os"
)

func printCorpusUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s corpus <subcommand> [options]\n\n", os.Args[0])
	fmt.Fprintf(w, "  Manage the corpus staging database.\n")
	fmt.Fprintf(w, "\nSubcommands:\n")
	fmt.Fprintf(w, helpCmdFmt, "ls", "List documents in the corpus staging database.")
}

func runCorpusCommand(args []string, setup *Setup, ui UI) error {
	if len(args) < 1 {
		printCorpusUsage(ui.Err)
		return fmt.Errorf("corpus requires a subcommand")
	}

	sub := args[0]
	subArgs := args[1:]

	if sub == "--help" || sub == "-h" {
		printCorpusUsage(ui.Out)
		return nil
	}

	switch sub {
	case "ls":
		opts, err := parseCorpusLsArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusLsCommand(repo, opts, ui)

	default:
		printCorpusUsage(ui.Err)
		return fmt.Errorf("unknown corpus subcommand: %s", sub)
	}
}
