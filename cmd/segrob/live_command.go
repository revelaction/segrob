package main

import (
	"fmt"
	"io"
	"os"
)

func printLiveUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s live <subcommand> [options]\n\n", os.Args[0])
	fmt.Fprintf(w, "  Manage the live production database.\n")
	fmt.Fprintf(w, "\nSubcommands:\n")
	fmt.Fprintf(w, helpCmdFmt, "ls", "List all documents in the repository.")
	fmt.Fprintf(w, helpCmdFmt, "show", "Show contents of a document file or DB entry.")
	fmt.Fprintf(w, helpCmdFmt, "query", "Enter interactive query mode.")
}

func runLiveCommand(args []string, setup *Setup, ui UI) error {
	if len(args) < 1 {
		printLiveUsage(ui.Err)
		return fmt.Errorf("live requires a subcommand")
	}

	sub := args[0]
	subArgs := args[1:]

	if sub == "--help" || sub == "-h" {
		printLiveUsage(ui.Out)
		return nil
	}

	switch sub {
	case "ls":
		opts, _, err := parseLiveLsArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return liveLsCommand(repo, opts, ui)

	case "show":
		opts, id, err := parseLiveShowArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewDocRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return liveShowCommand(repo, opts, id, ui)

	case "query":
		opts, _, _, err := parseLiveQueryArgs(subArgs, ui)
		if err != nil {
			return err
		}
		dr, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		tr, err := setup.NewTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return liveQueryCommand(dr, tr, opts, ui)

	default:
		printLiveUsage(ui.Err)
		return fmt.Errorf("unknown live subcommand: %s", sub)
	}
}
