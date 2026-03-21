package main

import (
	"fmt"
	"io"
	"os"
)

func printLiveUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s live <subcommand> [options]\n\n", os.Args[0])
	fmt.Fprintf(w, "  Manage the live production database.\n")

	fmt.Fprintf(w, "\nSubcommands: Doc - Sentences\n")
	fmt.Fprintf(w, helpCmdFmt, "ls", "List all documents in the repository.")
	fmt.Fprintf(w, helpCmdFmt, "show", "Show document contents or statistics.")
	fmt.Fprintf(w, helpCmdFmt, "show-sent", "Show sentence details or statistics.")
	fmt.Fprintf(w, helpCmdFmt, "unpublish", "Remove a document from all live tables by id.")
	fmt.Fprintf(w, helpCmdFmt, "find", "Find sentences matching a topic expression.")
	fmt.Fprintf(w, helpCmdFmt, "find-topics", "Show topics for a specific sentence.")
	fmt.Fprintf(w, helpCmdFmt, "query", "Enter interactive query mode.")

	fmt.Fprintf(w, "\nSubcommands: Topics\n")
	fmt.Fprintf(w, helpCmdFmt, "ls-topic", "List all unique topics in the repository.")
	fmt.Fprintf(w, helpCmdFmt, "show-topic", "Show expressions for a specific topic.")
	fmt.Fprintf(w, helpCmdFmt, "import-topic", "Import topics from filesystem to SQLite.")
	fmt.Fprintf(w, helpCmdFmt, "export-topic", "Export topics from SQLite to filesystem.")

	fmt.Fprintf(w, "\nSubcommands: Other\n")
	fmt.Fprintf(w, helpCmdFmt, "init", "Initialize a new SQLite database with the required schema.")
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
		pool, err := setup.GetPool(opts.DbPath)
		if err != nil {
			return err
		}
		return liveInitCommand(pool, opts, ui)

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

	case "import-topic":
		opts, err := parseLiveImportTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		return liveImportTopicCommand(opts, ui)

	case "export-topic":
		opts, err := parseLiveExportTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		return liveExportTopicCommand(opts, ui)

	default:
		printLiveUsage(ui.Err)
		return fmt.Errorf("unknown live subcommand: %s", sub)
	}
}
