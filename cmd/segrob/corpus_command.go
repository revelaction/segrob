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
	fmt.Fprintf(w, helpCmdFmt, "show", "Show rendered contents of a corpus document's NLP field.")
	fmt.Fprintf(w, helpCmdFmt, "dump-txt", "Output the txt field of a corpus document byte-exact.")
	fmt.Fprintf(w, helpCmdFmt, "dump-nlp", "Output the nlp field of a corpus document.")
	fmt.Fprintf(w, helpCmdFmt, "ingest-nlp", "Process document text with NLP and store in corpus.")
	fmt.Fprintf(w, helpCmdFmt, "ingest-meta", "Scan a directory for epub files and build a corpus database.")
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

	default:
		printCorpusUsage(ui.Err)
		return fmt.Errorf("unknown corpus subcommand: %s", sub)
	}
}
