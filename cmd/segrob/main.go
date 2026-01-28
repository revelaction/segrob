package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
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

	switch cmd {
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
		opts, arg, isFile, err := parseDocArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return docCommand(opts, arg, isFile, ui)

	case "sentence":
		source, sentId, isFile, err := parseSentenceArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return sentenceCommand(source, sentId, isFile, ui)

	case "topics":
		opts, source, sentId, isTopicFile, isSourceFile, err := parseTopicsArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return topicsCommand(opts, source, sentId, isTopicFile, isSourceFile, ui)

	case "expr":
		opts, cmdArgs, isDocFile, err := parseExprArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return exprCommand(opts, cmdArgs, isDocFile, ui)

	case "query":
		opts, isTopicFile, isDocFile, err := parseQueryArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return queryCommand(opts, isTopicFile, isDocFile, ui)

	case "edit":
		opts, isFile, err := parseEditArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return editCommand(opts, isFile, ui)

	case "topic":
		opts, name, isFile, err := parseTopicArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return topicCommand(opts, name, isFile, ui)

	case "stat":
		source, sentId, isFile, err := parseStatArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return statCommand(source, sentId, isFile, ui)

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

	case "import-doc":
		opts, err := parseImportDocArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return importDocCommand(opts, ui)

	case "export-doc":
		opts, err := parseExportDocArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return exportDocCommand(opts, ui)
	}

	return fmt.Errorf("unknown command: %s", cmd)
}

func getTopicHandler(path string, isFile bool) (storage.TopicRepository, error) {
	if isFile {
		pool, err := zombiezen.NewPool(path)
		if err != nil {
			return nil, err
		}
		return zombiezen.NewTopicHandler(pool), nil
	}

	return filesystem.NewTopicHandler(path), nil
}
