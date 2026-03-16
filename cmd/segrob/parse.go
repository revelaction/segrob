package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Column widths for all help output sections.
// Adjust these to change the layout globally.
const (
	helpCmdFmt = "  %-14s %s\n" // commands:  "corpus-meta    description"
	helpOptFmt = "  %-28s %s\n" // options:   "-d, --doc-path PATH    description"
	helpArgFmt = "  %-28s %s\n" // arguments: "doc_id                 description"
)

// printOpt prints one option line: flags + optional meta value + description.
// Example: printOpt(w, "-s, --start", "INDEX", "Index of first sentence (default: 0)")
func printOpt(w io.Writer, flags, meta, desc string) {
	left := flags
	if meta != "" {
		left = flags + " " + meta
	}
	fmt.Fprintf(w, helpOptFmt, left, desc)
}

// ShowOptions defines shared pagination and source settings for "show" commands (live and corpus).
type ShowOptions struct {
	Start  int
	Count  *int
	DbPath string // path to segrob.db or corpus.db
	Stats  bool   // -s/--stats: show document statistics
}

// stringSliceFlag implements flag.Value for multi-value strings
type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// enumFlag implements flag.Value for restricted strings
type enumFlag struct {
	allowed []string
	value   *string
}

func (e *enumFlag) String() string {
	if e.value == nil {
		return ""
	}
	return *e.value
}

func (e *enumFlag) Set(value string) error {
	for _, a := range e.allowed {
		if a == value {
			*e.value = value
			return nil
		}
	}
	return fmt.Errorf("allowed values are %s", strings.Join(e.allowed, ", "))
}

// optionalInt implements flag.Value for optional integer flags
type optionalInt struct {
	value *int
}

func (o *optionalInt) String() string {
	if o.value == nil {
		return ""
	}
	return strconv.Itoa(*o.value)
}

func (o *optionalInt) Set(s string) error {
	v, err := strconv.Atoi(s)
	if err != nil {
		return err
	}
	o.value = &v
	return nil
}

func parseMainArgs(args []string, ui UI) (string, []string, error) {
	fs := flag.NewFlagSet("segrob", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	setupUsage(fs)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return "", nil, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return "", nil, err
	}

	if fs.NArg() == 0 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return "", nil, errors.New("no command provided")
	}

	cmd := fs.Arg(0)
	cmdArgs := fs.Args()[1:]
	return cmd, cmdArgs, nil
}

func parseBashArgs(args []string, ui UI) error {
	fs := flag.NewFlagSet("bash", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s bash\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output bash completion script.\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return err
	}
	return nil
}

func parseCompleteArgs(args []string, ui UI) ([]string, error) {
	fs := flag.NewFlagSet("complete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	return fs.Args(), nil
}

func setupUsage(fs *flag.FlagSet) {
	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s command [command options] [arguments...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Sentence dictionary based on NLP topics\n")

		fmt.Fprintf(w, "\nCommands:\n")
		fmt.Fprintf(w, helpCmdFmt, "corpus", "Manage the corpus staging database.")
		fmt.Fprintf(w, helpCmdFmt, "live", "Manage the live production database.")
		fmt.Fprintf(w, helpCmdFmt, "bash", "Output bash completion script.")
		fmt.Fprintf(w, helpCmdFmt, "version", "Show version information.")
		fmt.Fprintf(w, helpCmdFmt, "help", "Show help for a command.")

		fmt.Fprintf(w, "\nVersion: %s, commit %s\n", BuildTag, BuildCommit)
	}
}

func validatePaths(path1, path2 string) error {
	if path1 == "" || path2 == "" {
		return nil
	}

	i1, err := os.Stat(path1)
	if err != nil {
		return nil // Let factory handle missing paths
	}
	i2, err := os.Stat(path2)
	if err != nil {
		return nil
	}

	if !i1.IsDir() && !i2.IsDir() {
		a1, _ := filepath.Abs(path1)
		a2, _ := filepath.Abs(path2)
		if a1 != a2 {
			return fmt.Errorf("using two different SQLite files is not supported: %s and %s", path1, path2)
		}
	}
	return nil
}
