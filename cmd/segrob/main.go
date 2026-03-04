package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
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

	// Centralized Terminal Reset
	//
	// The issue occurs because go-prompt puts your terminal into Raw Mode (to
	// handle custom keybinds and colors) but fails to restore it to Cooked
	// Mode (canonical mode) upon exit. When the terminal is left in Raw Mode,
	// it often disables local echo (typing is invisible) and carriage
	// returns.
	// For interactive commands, we save the terminal state (Cooked Mode)
	// and strictly restore it when the function returns.
	if cmd == "query" || cmd == "edit" {
		fd := int(os.Stdin.Fd())
		if state, err := term.GetState(fd); err == nil {
			defer term.Restore(fd, state)
		}
	}

	setup := NewSetup()
	defer setup.Close()

	switch cmd {
	case "version":
		return versionCommand(ui)

	case "help":
		if len(args) > 0 {
			return runCommand(args[0], []string{"--help"}, ui)
		}
		fs := flag.NewFlagSet("segrob", flag.ContinueOnError)
		fs.SetOutput(ui.Out)
		setupUsage(fs)
		fs.Usage()
		return nil

	case "doc":
		opts, id, err := parseDocArgs(args, ui)
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
		return docCommand(repo, opts, id, ui)

	case "ls-doc":
		opts, _, err := parseLsDocArgs(args, ui)
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
		return lsDocCommand(repo, opts, ui)

	case "ls-labels":
		opts, err := parseLsLabelsArgs(args, ui)
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
		return lsLabelsCommand(repo, opts, ui)

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

	case "query":
		opts, _, _, err := parseQueryArgs(args, ui)
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
		return queryCommand(dr, tr, opts, ui)

	case "edit":
		opts, _, err := parseEditArgs(args, ui)
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
		return editCommand(tr, opts, ui)

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

	case "live":
		opts, err := parseLiveArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
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
		return liveCommand(corpusRepo, docRepo, opts, ui)

	case "corpus-nlp":
		opts, err := parseCorpusNlp(args)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		corpusRepo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusNlpCommand(corpusRepo, opts, ui)

	case "add-label":
		opts, err := parseAddLabelArgs(args, ui)
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
		return addLabelCommand(repo, opts, ui)

	case "remove-label":
		opts, err := parseRemoveLabelArgs(args, ui)
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
		return removeLabelCommand(repo, opts, ui)

	case "corpus-meta":
		opts, err := parseCorpusMetaArgs(args, ui)
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
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return corpusMetaCommand(pool, repo, opts, ui)

	case "cat-txt":
		opts, err := parseCatTxtArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		repo, err := setup.NewCorpusRepository(opts.DbPath)
		if err != nil {
			return err
		}
		return catTxtCommand(repo, opts, ui)
	}

	return fmt.Errorf("unknown command: %s", cmd)
}
