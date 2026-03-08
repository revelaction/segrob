package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

var (
	BuildTag    string
	BuildCommit string
)

// UI contains the output streams for the application.
// Used for injecting buffers during testing.
type UI struct {
	Out io.Writer
	Err io.Writer
}

func main() {
	ui := UI{Out: os.Stdout, Err: os.Stderr}

	cmd, args, err := parseMainArgs(os.Args[1:], ui)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if err := runCommand(cmd, args, ui); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fprintErr(ui.Err, err)
		os.Exit(1)
	}

	os.Exit(0)
}

func fprintErr(w io.Writer, err error) {
	_, _ = fmt.Fprintf(w, "segrob: %v\n", err)
}

func runCommand(cmd string, args []string, ui UI) error {

	setup := NewSetup()
	defer setup.Close()

	switch cmd {
	case "version":
		return versionCommand(ui)

	case "help":
		if len(args) > 0 {
			return runCommand(args[0], append(args[1:], "--help"), ui)
		}
		fs := flag.NewFlagSet("segrob", flag.ContinueOnError)
		fs.SetOutput(ui.Out)
		setupUsage(fs)
		fs.Usage()
		return nil

	case "live":
		return runLiveCommand(args, setup, ui)

	case "sentence":
		opts, docId, sentId, err := parseSentenceArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		repo, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return sentenceCommand(repo, opts, docId, sentId, ui)

	case "topics":
		opts, docId, sentId, err := parseTopicsArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
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
		return topicsCommand(dr, tr, opts, docId, sentId, ui)

	case "expr":
		opts, cmdArgs, _, err := parseExprArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		dr, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return exprCommand(dr, opts, cmdArgs, ui)

	case "topic":
		opts, name, _, err := parseTopicArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		tr, err := setup.NewTopicRepository(opts.TopicPath)
		if err != nil {
			return err
		}
		return topicCommand(tr, opts, name, ui)

	case "stat":
		opts, docId, sentId, err := parseStatArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		repo, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return statCommand(repo, opts, docId, sentId, ui)

	case "bash":
		if err := parseBashArgs(args, ui); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return bashCommand(ui)

	case "complete":
		completeArgs, err := parseCompleteArgs(args, ui)
		if err != nil {
			return err
		}
		return completeCommand(completeArgs, ui)

	case "import-topic":
		opts, err := parseImportTopicArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return importTopicCommand(opts, ui)

	case "export-topic":
		opts, err := parseExportTopicArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return exportTopicCommand(opts, ui)

	case "init-db":
		opts, err := parseInitDbArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		pool, err := setup.GetPool(opts.DbPath)
		if err != nil {
			return err
		}
		return initDbCommand(pool, opts, ui)

	case "corpus":
		return runCorpusCommand(args, setup, ui)

	case "label-rm":
		opts, err := parseLabelRmArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		repo, err := setup.NewDocRepository(opts.DocPath)
		if err != nil {
			return err
		}
		return labelRmCommand(repo, opts, ui)
	}

	return fmt.Errorf("unknown command: %s", cmd)
}
