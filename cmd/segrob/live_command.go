package main

import (
	"fmt"
	"io"
	"os"
)

func printLiveUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Usage: %s live <subcommand> [options]\n\n", os.Args[0])
	_, _ = fmt.Fprintf(w, "  Manage the live production database.\n")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Doc - Sentences\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ls", "List all documents in the repository.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "show", "Show document contents or statistics.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "show-sent", "Show sentence details or statistics.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "unpublish", "Remove a document from all live tables by id.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "find", "Find sentences matching a topic expression.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "find-topics", "Show topics for a specific sentence.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "query", "Enter interactive query mode.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Topics\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ls-topic", "List all unique topics in the repository.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "show-topic", "Show expressions for a specific topic.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Other\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "init", "Initialize a new SQLite database with the required schema.")
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

	case "ls-topic":
		opts, _, err := parseLiveLsTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewLiveTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return liveLsTopicCommand(repo, opts, ui)

	case "show-topic":
		opts, name, _, err := parseLiveShowTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewLiveTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return liveShowTopicCommand(repo, opts, name, ui)

	case "find":
		opts, cmdArgs, _, err := parseLiveFindArgs(subArgs, ui)
		if err != nil {
			return err
		}
		dr, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return liveFindCommand(dr, opts, cmdArgs, ui)

	case "find-topics":
		opts, docId, sentId, err := parseLiveFindTopicsArgs(subArgs, ui)
		if err != nil {
			return err
		}
		dr, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		tr, err := setup.NewLiveTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return liveFindTopicsCommand(dr, tr, opts, docId, sentId, ui)

	case "init":
		opts, err := parseLiveInitArgs(subArgs, ui)
		if err != nil {
			return err
		}
		mgr, err := setup.NewSchemaManager(opts.DbPath)
		if err != nil {
			return err
		}
		return liveInitCommand(mgr, opts, ui)

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

	case "show-sent":
		opts, docId, sentId, err := parseLiveShowSentArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewDocRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return liveShowSentCommand(repo, opts, docId, sentId, ui)

	case "unpublish":
		opts, err := parseLiveUnpublishArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewDocRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return liveUnpublishCommand(repo, opts, ui)

	case "query":
		opts, _, _, err := parseLiveQueryArgs(subArgs, ui)
		if err != nil {
			return err
		}
		dr, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		tr, err := setup.NewLiveTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return liveQueryCommand(dr, tr, opts, ui)

	default:
		printLiveUsage(ui.Err)
		return fmt.Errorf("unknown live subcommand: %s", sub)
	}
}
