package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/revelaction/segrob/render"
)

// Option structs for subcommands that have flags
type LiveFindOptions struct {
	Labels    []string
	NoColor   bool
	NoPrefix  bool
	NMatches  int
	Format    string
	DocPath   string
	TopicPath string
	JSON      bool // output results as JSON
	Limit     int  // max matched results (0 = unlimited)
}

type LiveQueryOptions struct {
	Labels    []string
	NoColor   bool
	NoPrefix  bool
	NMatches  int
	Format    string
	TopicPath string
	DocPath   string
}

type LiveFindTopicsOptions struct {
	Format    string
	TopicPath string
	DocPath   string
}

type LiveLsTopicOptions struct {
	TopicPath string
}

type LiveShowTopicOptions struct {
	TopicPath string
}

type LiveLsOptions struct {
	DocPath string
	Match   string
}

type LiveShowSentOptions struct {
	DbPath string
	Stats  bool // -s/--stats: show sentence statistics
}

type LiveInitOptions struct {
	DbPath string
}

type LiveUnpublishOptions struct {
	DbPath string // --db / SEGROB_LIVE_DB
	ID     string // positional arg: document id
}

func parseLiveShowArgs(args []string, ui UI) (ShowOptions, string, error) {
	fs := flag.NewFlagSet("live show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ShowOptions
	fs.IntVar(&opts.Start, "start", 0, "")
	fs.BoolVar(&opts.Stats, "stats", false, "")
	fs.BoolVar(&opts.Stats, "s", false, "")

	var countOpt optionalInt
	fs.Var(&countOpt, "number", "")
	fs.Var(&countOpt, "n", "")

	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_LIVE_DB"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live show [options] <doc_id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show contents of a document from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document to show")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--start", "INDEX", "Index of the first sentence to show (default: 0)")
		printOpt(w, "-n, --number", "N", "Number of sentences to show")
		printOpt(w, "-s, --stats", "", "Show document statistics")
		printOpt(w, "--db", "FILE", "Path to docs directory or SQLite file (or SEGROB_LIVE_DB)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", err
		}
		return opts, "", err
	}

	opts.Count = countOpt.value

	if opts.DbPath == "" {
		return opts, "", errors.New("document source must be specified via --db or SEGROB_LIVE_DB")
	}

	if fs.NArg() != 1 {
		return opts, "", errors.New("live show requires exactly one argument: <id>")
	}
	id := fs.Arg(0)

	return opts, id, nil
}

func parseLiveLsArgs(args []string, ui UI) (LiveLsOptions, bool, error) {
	fs := flag.NewFlagSet("live ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveLsOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live ls [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all documents in the repository.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_LIVE_DB)")
		printOpt(w, "-m, --match", "STRING", "Only list documents with at least one label containing STRING")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, false, err
	}

	if opts.DocPath == "" {
		return opts, false, errors.New("no document source specified (use -d or SEGROB_LIVE_DB)")
	}

	info, err := os.Stat(opts.DocPath)
	if err != nil {
		return opts, false, fmt.Errorf("document source not found: %s", opts.DocPath)
	}

	return opts, info.IsDir(), nil
}

func parseLiveShowSentArgs(args []string, ui UI) (LiveShowSentOptions, string, int, error) {
	fs := flag.NewFlagSet("live show-sent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveShowSentOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.BoolVar(&opts.Stats, "stats", false, "")
	fs.BoolVar(&opts.Stats, "s", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live show-sent [options] <doc_id> <sentence_id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show details of a specific sentence from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "sentence_id", "Index of the sentence")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --stats", "", "Show sentence statistics")
		printOpt(w, "--db", "PATH", "Path to SQLite file (or SEGROB_LIVE_DB)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", 0, err
		}
		return opts, "", 0, err
	}

	if opts.DbPath == "" {
		return opts, "", 0, errors.New("document source must be specified via --db or SEGROB_LIVE_DB")
	}

	if fs.NArg() != 2 {
		return opts, "", 0, errors.New("live show-sent requires exactly two arguments: <doc_id> <sentence_id>")
	}

	docId := fs.Arg(0)

	sentId, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		return opts, "", 0, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
	}

	return opts, docId, sentId, nil
}

func parseLiveFindTopicsArgs(args []string, ui UI) (LiveFindTopicsOptions, string, int, error) {
	fs := flag.NewFlagSet("live find-topics", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveFindTopicsOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_LIVE_DB"), "")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "")
	fs.Var(formatFlag, "f", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live find-topics [options] <doc_id> <sentence_id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show topics for a specific sentence from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "sentence_id", "Index of the sentence")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_LIVE_DB)")
		printOpt(w, "-t, --topic-path", "PATH", "Path to topics directory or SQLite file (or SEGROB_TOPIC_PATH)")
		printOpt(w, "-f, --format", "FORMAT", "Output format: all, part, or lemma (default: "+render.Defaultformat+")")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", 0, err
		}
		return opts, "", 0, err
	}

	if opts.TopicPath == "" {
		return opts, "", 0, errors.New("topic source must be specified via -t or SEGROB_TOPIC_PATH")
	}

	if opts.DocPath == "" {
		return opts, "", 0, errors.New("document source must be specified via -d or SEGROB_LIVE_DB")
	}

	if fs.NArg() != 2 {
		return opts, "", 0, errors.New("find-topics command needs exactly two arguments: <doc_id> <sentence_id>")
	}

	docId := fs.Arg(0)

	sentId, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		return opts, "", 0, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
	}

	if err := validatePaths(opts.DocPath, opts.TopicPath); err != nil {
		return opts, "", 0, err
	}

	return opts, docId, sentId, nil
}

func parseLiveFindArgs(args []string, ui UI) (LiveFindOptions, []string, bool, error) {
	fs := flag.NewFlagSet("live find", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveFindOptions
	labels := (*stringSliceFlag)(&opts.Labels)
	fs.Var(labels, "label", "")
	fs.Var(labels, "l", "")

	fs.BoolVar(&opts.NoColor, "no-color", false, "")
	fs.BoolVar(&opts.NoColor, "c", false, "")

	fs.BoolVar(&opts.NoPrefix, "no-prefix", false, "")
	fs.BoolVar(&opts.NoPrefix, "x", false, "")

	fs.IntVar(&opts.NMatches, "nmatches", 0, "")
	fs.IntVar(&opts.NMatches, "n", 0, "")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "")
	fs.Var(formatFlag, "f", "")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_LIVE_DB"), "")

	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.BoolVar(&opts.JSON, "json", false, "")
	fs.BoolVar(&opts.JSON, "j", false, "")

	fs.IntVar(&opts.Limit, "limit", 0, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live find [options] <expr>...\n\n", os.Args[0])
		fmt.Fprintf(w, "  Find sentences matching a topic expression.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "expr", "One or more topic expression items")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_LIVE_DB)")
		printOpt(w, "-t, --topic-path", "PATH", "Path to topics directory or SQLite file (or SEGROB_TOPIC_PATH)")
		printOpt(w, "-l, --label", "LABEL", "Only scan documents matching this label (repeatable, all required)")
		printOpt(w, "-f, --format", "FORMAT", "Output format: all, part, or lemma (default: "+render.Defaultformat+")")
		printOpt(w, "-n, --nmatches", "N", "Only show sentences with match score greater than N (default: 0)")
		printOpt(w, "--limit", "N", "Maximum number of results to return (default: 0 = unlimited)")
		printOpt(w, "-c, --no-color", "", "Disable color formatting in output")
		printOpt(w, "-x, --no-prefix", "", "Omit metadata prefixes from output")
		printOpt(w, "-j, --json", "", "Output results as JSON")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, nil, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, nil, false, err
	}

	if fs.NArg() < 1 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return opts, nil, false, errors.New("find command needs at least one argument")
	}

	if opts.DocPath == "" {
		return opts, nil, false, errors.New("Doc path must be specified via -d or SEGROB_LIVE_DB")
	}

	info, err := os.Stat(opts.DocPath)
	if err != nil {
		return opts, nil, false, fmt.Errorf("Doc path not found: %s", opts.DocPath)
	}

	return opts, fs.Args(), !info.IsDir(), nil
}

func parseLiveQueryArgs(args []string, ui UI) (LiveQueryOptions, bool, bool, error) {
	fs := flag.NewFlagSet("live query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveQueryOptions
	labels := (*stringSliceFlag)(&opts.Labels)
	fs.Var(labels, "label", "")
	fs.Var(labels, "l", "")

	fs.BoolVar(&opts.NoColor, "no-color", false, "")
	fs.BoolVar(&opts.NoColor, "c", false, "")

	fs.BoolVar(&opts.NoPrefix, "no-prefix", false, "")
	fs.BoolVar(&opts.NoPrefix, "x", false, "")

	fs.IntVar(&opts.NMatches, "nmatches", 0, "")
	fs.IntVar(&opts.NMatches, "n", 0, "")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "")
	fs.Var(formatFlag, "f", "")

	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_LIVE_DB"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_LIVE_DB"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live query [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Enter interactive query mode.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_LIVE_DB)")
		printOpt(w, "-t, --topic-path", "PATH", "Path to topics directory or SQLite file (or SEGROB_TOPIC_PATH)")
		printOpt(w, "-l, --label", "LABEL", "Only scan documents matching this label (repeatable, all required)")
		printOpt(w, "-f, --format", "FORMAT", "Output format: all, part, or lemma (default: "+render.Defaultformat+")")
		printOpt(w, "-n, --nmatches", "N", "Only show sentences with match score greater than N (default: 0)")
		printOpt(w, "-c, --no-color", "", "Disable color formatting in output")
		printOpt(w, "-x, --no-prefix", "", "Omit metadata prefixes from output")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, false, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, false, false, err
	}

	if opts.TopicPath == "" {
		return opts, false, false, errors.New("Topic path must be specified via -t or SEGROB_TOPIC_PATH")
	}

	if opts.DocPath == "" {
		return opts, false, false, errors.New("Doc path must be specified via -d or SEGROB_LIVE_DB")
	}

	tinfo, err := os.Stat(opts.TopicPath)
	if err != nil {
		return opts, false, false, fmt.Errorf("Topic path not found: %s", opts.TopicPath)
	}

	dinfo, err := os.Stat(opts.DocPath)
	if err != nil {
		return opts, false, false, fmt.Errorf("Doc path not found: %s", opts.DocPath)
	}

	if err := validatePaths(opts.DocPath, opts.TopicPath); err != nil {
		return opts, false, false, err
	}

	return opts, !tinfo.IsDir(), !dinfo.IsDir(), nil
}

func parseLiveLsTopicArgs(args []string, ui UI) (LiveLsTopicOptions, bool, error) {
	fs := flag.NewFlagSet("live ls-topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveLsTopicOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live ls-topic [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all topic names in the repository.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-t, --topic-path", "PATH", "Path to topics directory or SQLite file (or SEGROB_TOPIC_PATH)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, false, err
	}

	if opts.TopicPath == "" {
		return opts, false, errors.New("Topic path must be specified via -t or SEGROB_TOPIC_PATH")
	}

	info, err := os.Stat(opts.TopicPath)
	if err != nil {
		return opts, false, fmt.Errorf("Topic path not found: %s", opts.TopicPath)
	}

	return opts, !info.IsDir(), nil
}

func parseLiveShowTopicArgs(args []string, ui UI) (LiveShowTopicOptions, string, bool, error) {
	fs := flag.NewFlagSet("live show-topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveShowTopicOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live show-topic [options] <name>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show expressions of a named topic.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "name", "Topic name to inspect")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-t, --topic-path", "PATH", "Path to topics directory or SQLite file (or SEGROB_TOPIC_PATH)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, "", false, err
	}

	if opts.TopicPath == "" {
		return opts, "", false, errors.New("Topic path must be specified via -t or SEGROB_TOPIC_PATH")
	}

	if fs.NArg() != 1 {
		return opts, "", false, errors.New("live show-topic requires exactly one argument: <name>")
	}
	name := fs.Arg(0)

	info, err := os.Stat(opts.TopicPath)
	if err != nil {
		return opts, "", false, fmt.Errorf("Topic path not found: %s", opts.TopicPath)
	}

	return opts, name, !info.IsDir(), nil
}

func parseLiveInitArgs(args []string, ui UI) (LiveInitOptions, error) {
	fs := flag.NewFlagSet("live init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveInitOptions

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live init <db>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Initialize a new SQLite database with the required schema.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "db", "Path to the SQLite file to create")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if fs.NArg() != 1 {
		return opts, errors.New("init command requires exactly one argument: <db>")
	}

	opts.DbPath = fs.Arg(0)
	return opts, nil
}

func parseLiveUnpublishArgs(args []string, ui UI) (LiveUnpublishOptions, error) {
	fs := flag.NewFlagSet("live unpublish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveUnpublishOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_LIVE_DB"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live unpublish [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Remove a document from all live tables.\n")
		fmt.Fprintf(w, "  The removal is the reverse of publish: the live switch (lemma index) is\n")
		fmt.Fprintf(w, "  cut first, then labels, sentences, and finally the doc row.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to unpublish")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "PATH", "Target segrob SQLite file (or SEGROB_LIVE_DB)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if fs.NArg() != 1 {
		return opts, errors.New("live unpublish requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("document source must be specified via --db or SEGROB_LIVE_DB")
	}

	return opts, nil
}
