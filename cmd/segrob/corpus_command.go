package main

import (
	"fmt"
	"io"
	"os"
)

func printCorpusUsage(w io.Writer) {
	fmt.Fprintf(w, "Usage: %s corpus <subcommand> [options]\n\n", os.Args[0])
	fmt.Fprintf(w, "  Manage the corpus staging database.\n")

	fmt.Fprintf(w, "\nSubcommands: Documents\n")
	fmt.Fprintf(w, helpCmdFmt, "ls", "List documents in the corpus staging database.")
	fmt.Fprintf(w, helpCmdFmt, "show", "Show rendered contents of a document's NLP field.")
	fmt.Fprintf(w, helpCmdFmt, "ack", "Acknowledge a corpus document text or NLP.")
	fmt.Fprintf(w, helpCmdFmt, "rm", "Remove a document from the corpus.")

	fmt.Fprintf(w, "\nSubcommands: Dump\n")
	fmt.Fprintf(w, helpCmdFmt, "dump-txt", "Output the txt field of a corpus document byte-exact.")
	fmt.Fprintf(w, helpCmdFmt, "dump-nlp", "Output the nlp field of a corpus document.")

	fmt.Fprintf(w, "\nSubcommands: Ingest\n")
	fmt.Fprintf(w, helpCmdFmt, "ingest-nlp", "Process document text with NLP and store in corpus.")
	fmt.Fprintf(w, helpCmdFmt, "ingest-meta", "Scan a directory for epub files and build a corpus database.")
	fmt.Fprintf(w, helpCmdFmt, "push-txt", "Update a corpus document text from a file.")

	fmt.Fprintf(w, "\nSubcommands: Publish\n")
	fmt.Fprintf(w, helpCmdFmt, "publish", "Move document(s) from corpus to live (all ACKed when no id).")
	fmt.Fprintf(w, helpCmdFmt, "publish-label", "Push corpus labels into live tables for a document.")

	fmt.Fprintf(w, "\nSubcommands: Labels\n")
	fmt.Fprintf(w, helpCmdFmt, "ls-label", "List all unique labels in the corpus.")
	fmt.Fprintf(w, helpCmdFmt, "set-label", "Add or remove labels from a corpus document.")

	fmt.Fprintf(w, "\nSubcommands: Topics\n")
	fmt.Fprintf(w, helpCmdFmt, "ls-topic", "List all unique topics in the repository.")
	fmt.Fprintf(w, helpCmdFmt, "show-topic", "Show expressions for a specific topic.")
	fmt.Fprintf(w, helpCmdFmt, "edit", "Enter interactive edit mode.")
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

	case "show":
		opts, id, err := parseCorpusShowArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusShowCommand(repo, opts, id, ui)

	case "ack":
		opts, err := parseCorpusAckArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusAckCommand(repo, opts, ui)

	case "rm":
		opts, err := parseCorpusRmArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusRmCommand(repo, opts, ui)

	case "publish":
		opts, err := parseCorpusPublishArgs(subArgs, ui)
		if err != nil {
			return err
		}
		corpusRepo, err := setup.NewCorpusRepository(opts.From)
		if err != nil {
			return err
		}
		docRepo, err := setup.NewDocRepository(opts.To)
		if err != nil {
			return err
		}
		return corpusPublishCommand(corpusRepo, docRepo, opts, ui)

	case "publish-label":
		opts, err := parseCorpusPublishLabelArgs(subArgs, ui)
		if err != nil {
			return err
		}
		corpusRepo, err := setup.NewCorpusRepository(opts.From)
		if err != nil {
			return err
		}
		docRepo, err := setup.NewDocRepository(opts.To)
		if err != nil {
			return err
		}
		return corpusPublishLabelCommand(corpusRepo, docRepo, opts, ui)

	case "dump-txt":
		opts, err := parseCorpusDumpTxtArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusDumpTxtCommand(repo, opts, ui)

	case "dump-nlp":
		opts, err := parseCorpusDumpNlpArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusDumpNlpCommand(repo, opts, ui)

	case "ingest-nlp":
		opts, err := parseCorpusIngestNlpArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusIngestNlpCommand(repo, opts, ui)

	case "ingest-meta":
		opts, err := parseCorpusIngestMetaArgs(subArgs, ui)
		if err != nil {
			return err
		}
		pool, err := setup.GetPool(opts.DbPath)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusIngestMetaCommand(pool, repo, opts, ui)

	case "push-txt":
		opts, err := parseCorpusPushTxtArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusPushTxtCommand(repo, opts, ui)

	case "ls-label":
		opts, err := parseCorpusLsLabelArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusLsLabelCommand(repo, opts, ui)

	case "set-label":
		opts, err := parseCorpusSetLabelArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusSetLabelCommand(repo, opts, ui)

	case "ls-topic":
		opts, err := parseCorpusLsTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusLsTopicCommand(repo, opts, ui)

	case "show-topic":
		opts, name, err := parseCorpusShowTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusShowTopicCommand(repo, opts, name, ui)

	case "edit":
		opts, err := parseCorpusEditArgs(subArgs, ui)
		if err != nil {
			return err
		}
		repo, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusEditCommand(repo, opts, ui)

	default:
		printCorpusUsage(ui.Err)
		return fmt.Errorf("unknown corpus subcommand: %s", sub)
	}
}
