package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func printCorpusUsage(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Usage: %s corpus <subcommand> [options]\n\n", os.Args[0])
	_, _ = fmt.Fprintf(w, "  Manage the corpus staging database.\n")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Init\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "init", "Initialize the corpus staging database.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Documents\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ls", "List documents in the corpus staging database.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "show", "Show rendered contents of a document's NLP field.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ack", "Acknowledge a corpus document text or NLP.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "rm", "Remove a document from the corpus.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Dump\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "dump-txt", "Output the txt field of a corpus document byte-exact.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "dump-nlp", "Output the nlp field of a corpus document.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Ingest\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ingest-nlp", "Process document text with NLP and store in corpus.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ingest-meta", "Scan a directory for epub files and build a corpus database.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "push-txt", "Update a corpus document text from a file.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Publish\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "publish", "Move document(s) from corpus to live (all ACKed when no id).")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "publish-label", "Push corpus labels into live tables for a document.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "publish-topic", "Copy all topics from the corpus database to the live database.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Backup\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "backup", "Create a gzipped backup of the corpus database.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Labels\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ls-label", "List all unique labels in the corpus.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "set-label", "Add or remove labels from a corpus document.")

	_, _ = fmt.Fprintf(w, "\nSubcommands: Topics\n")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "ls-topic", "List all unique topics in the repository.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "show-topic", "Show expressions for a specific topic.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "import-topic", "Import topics from filesystem to SQLite.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "export-topic", "Export topics from SQLite to filesystem.")
	_, _ = fmt.Fprintf(w, helpCmdFmt, "edit", "Enter interactive edit mode.")
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
	case "init":
		opts, err := parseCorpusInitArgs(subArgs, ui)
		if err != nil {
			return err
		}
		mgr, err := setup.NewSchemaManager(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusInitCommand(mgr, opts, ui)

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

	case "publish-topic":
		opts, err := parseCorpusPublishTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		corpusTopics, err := setup.NewCorpusTopicRepository(opts.From)
		if err != nil {
			return err
		}
		liveTopics, err := setup.NewLiveTopicRepository(opts.To)
		if err != nil {
			return err
		}
		return corpusPublishTopicCommand(corpusTopics, liveTopics, opts, ui)

	case "backup":
		opts, err := parseCorpusBackupArgs(subArgs, ui)
		if err != nil {
			return err
		}
		srcRepo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		srcTopicsRepo, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}

		// Temp SQLite file for backup creation
		tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("corpus-backup-%d.db", time.Now().UnixNano()))

		dstMgr, err := setup.NewSchemaManager(tempPath, "_journal_mode=DELETE")
		if err != nil {
			return err
		}
		dstRepo, err := setup.NewCorpusRepository(tempPath)
		if err != nil {
			return err
		}
		dstTopicsRepo, err := setup.NewCorpusTopicRepository(tempPath)
		if err != nil {
			return err
		}

		err = corpusBackupCommand(srcRepo, srcTopicsRepo, dstMgr, dstRepo, dstTopicsRepo, tempPath, opts, ui)
		if err != nil {
			return err
		}

		return nil

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
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusIngestMetaCommand(repo, opts, ui)

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

	case "import-topic":
		opts, err := parseCorpusImportTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		dst, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusImportTopicCommand(dst, opts, ui)

	case "export-topic":
		opts, err := parseCorpusExportTopicArgs(subArgs, ui)
		if err != nil {
			return err
		}
		src, err := setup.NewCorpusTopicRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusExportTopicCommand(src, opts, ui)

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
