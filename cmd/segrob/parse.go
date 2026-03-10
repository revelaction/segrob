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

	"github.com/revelaction/segrob/render"
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

// ShowOptions defines shared pagination and source settings for "show" commands (live and corpus).
type ShowOptions struct {
	Start  int
	Count  *int
	DbPath string // path to segrob.db or corpus.db
	Stats  bool   // -s/--stats: show document statistics
}

type LiveLsOptions struct {
	DocPath string
	Match   string
}

type LiveLsLabelOptions struct {
	DbPath string
	Match  string
}

type LiveShowSentOptions struct {
	DbPath string
	Stats  bool // -s/--stats: show sentence statistics
}

type LiveEditOptions struct {
	TopicPath string
}

type ImportTopicOptions struct {
	From string
	To   string
}

type ExportTopicOptions struct {
	From string
	To   string
}

type LiveInitOptions struct {
	DbPath string
}

type CorpusIngestNlpOptions struct {
	NlpScript string
	DbPath    string // corpus db path
	ID        string
}

type CorpusPublishOptions struct {
	From string // corpus.db path (--from / SEGROB_CORPUS_PATH)
	To   string // segrob.db path (--to / SEGROB_DOC_PATH)
	ID   string // positional arg: document id
	Move bool   // -m/--move: delete nlp from corpus after success
}

func parseCorpusPublishArgs(args []string, ui UI) (CorpusPublishOptions, error) {
	fs := flag.NewFlagSet("corpus publish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusPublishOptions
	fs.StringVar(&opts.From, "from", os.Getenv("SEGROB_CORPUS_PATH"), "Source corpus SQLite file")
	fs.StringVar(&opts.To, "to", os.Getenv("SEGROB_DOC_PATH"), "Target segrob SQLite file")
	fs.BoolVar(&opts.Move, "move", false, "Delete nlp data from corpus after successful live")
	fs.BoolVar(&opts.Move, "m", false, "alias for -move")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus publish [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Move a document from corpus staging to live production tables.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to move to production")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--from", "PATH", "Source corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "--to", "PATH", "Target segrob SQLite file (or SEGROB_DOC_PATH)")
		printOpt(w, "-m, --move", "", "Delete NLP data from corpus after successful live")
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
		return opts, errors.New("corpus publish requires exactly one argument: <id>")
	}
	opts.ID = fs.Arg(0)

	if opts.From == "" {
		return opts, errors.New("corpus source must be specified via --from or SEGROB_CORPUS_PATH")
	}
	if opts.To == "" {
		return opts, errors.New("target db must be specified via --to or SEGROB_DOC_PATH")
	}

	return opts, nil
}

type LiveSetLabelOptions struct {
	DocID  string
	Labels []string
	DbPath string
	Delete bool
}

type CorpusIngestMetaOptions struct {
	DbPath string
	Dir    string
	Pandoc bool
}

type CorpusDumpTxtOptions struct {
	DbPath string // --db / SEGROB_CORPUS_PATH
	Output string // --output file path (empty = stdout)
	ID     string // positional arg: document id
}

type CorpusDumpNlpOptions struct {
	DbPath   string // --db / SEGROB_CORPUS_PATH
	NoLemmas bool   // -n, --no-lemmas
	Output   string // --output file path (empty = stdout)
	ID       string // positional arg: document id
}

type CorpusLsOptions struct {
	DbPath  string // --db / SEGROB_CORPUS_PATH
	Filter  string // optional positional filter
	WithNlp bool   // --with-nlp / -n
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

	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_DOC_PATH"), "")

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
		printOpt(w, "--db", "FILE", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, "", errors.New("document source must be specified via --db or SEGROB_DOC_PATH")
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
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live ls [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all documents in the repository.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, false, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	info, err := os.Stat(opts.DocPath)
	if err != nil {
		return opts, false, fmt.Errorf("document source not found: %s", opts.DocPath)
	}

	return opts, info.IsDir(), nil
}

func parseLiveLsLabelArgs(args []string, ui UI) (LiveLsLabelOptions, error) {
	fs := flag.NewFlagSet("live ls-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveLsLabelOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live ls-label [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all unique labels in the repository.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
		printOpt(w, "-m, --match", "STRING", "Only list labels containing STRING")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, err
	}

	if opts.DbPath == "" {
		return opts, errors.New("no document source specified (use --db or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseLiveShowSentArgs(args []string, ui UI) (LiveShowSentOptions, string, int, error) {
	fs := flag.NewFlagSet("live show-sent", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveShowSentOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_DOC_PATH"), "")
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
		printOpt(w, "--db", "PATH", "Path to SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, "", 0, errors.New("document source must be specified via --db or SEGROB_DOC_PATH")
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

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

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
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, "", 0, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
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

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

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
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, nil, false, errors.New("Doc path must be specified via -d or SEGROB_DOC_PATH")
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

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live query [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Enter interactive query mode.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, false, false, errors.New("Doc path must be specified via -d or SEGROB_DOC_PATH")
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

func parseLiveEditArgs(args []string, ui UI) (LiveEditOptions, bool, error) {
	fs := flag.NewFlagSet("live edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveEditOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live edit [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Enter interactive edit mode.\n")
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

func parseImportTopicArgs(args []string, ui UI) (ImportTopicOptions, error) {
	fs := flag.NewFlagSet("import-topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ImportTopicOptions
	fs.StringVar(&opts.From, "from", "", "")
	fs.StringVar(&opts.To, "to", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s import-topic --from <dir> --to <sqlite_file>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Import topics from a JSON directory into a SQLite database.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--from", "DIR", "Source directory containing JSON topic files")
		printOpt(w, "--to", "FILE", "Target SQLite database file")
	}

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if opts.From == "" || opts.To == "" {
		return opts, errors.New("--from and --to are required")
	}

	return opts, nil
}

func parseExportTopicArgs(args []string, ui UI) (ExportTopicOptions, error) {
	fs := flag.NewFlagSet("export-topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ExportTopicOptions
	fs.StringVar(&opts.From, "from", "", "")
	fs.StringVar(&opts.To, "to", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s export-topic --from <sqlite_file> --to <dir>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Export topics from a SQLite database to a JSON directory.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--from", "FILE", "Source SQLite database file")
		printOpt(w, "--to", "DIR", "Target directory for JSON topic files")
	}

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if opts.From == "" || opts.To == "" {
		return opts, errors.New("--from and --to are required")
	}

	return opts, nil
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

func parseCorpusIngestNlpArgs(args []string, ui UI) (CorpusIngestNlpOptions, error) {
	fs := flag.NewFlagSet("corpus ingest-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusIngestNlpOptions
	fs.StringVar(&opts.NlpScript, "nlp-script", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.NlpScript, "s", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ingest-nlp [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Process document text with NLP and store results in the corpus.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to process")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --nlp-script", "PATH", "Path to the Python NLP script (or SEGROB_NLP_SCRIPT)")
		printOpt(w, "--db", "FILE", "Path to the corpus SQLite database (or SEGROB_CORPUS_PATH)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if opts.NlpScript == "" {
		return opts, fmt.Errorf("--nlp-script must be supplied if SEGROB_NLP_SCRIPT is not set")
	}
	if opts.DbPath == "" {
		return opts, fmt.Errorf("--db must be supplied if SEGROB_CORPUS_PATH is not set")
	}

	if fs.NArg() != 1 {
		return opts, fmt.Errorf("requires exactly 1 argument (doc ID)")
	}
	opts.ID = fs.Arg(0)

	return opts, nil
}

func parseLiveSetLabelArgs(args []string, ui UI) (LiveSetLabelOptions, error) {
	fs := flag.NewFlagSet("live set-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveSetLabelOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.BoolVar(&opts.Delete, "delete", false, "")
	fs.BoolVar(&opts.Delete, "d", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live set-label [options] <doc_id> <label> [<label>...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Add or remove one or more labels from a document.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "label", "One or more labels to add/remove")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "PATH", "Path to SQLite file (or SEGROB_DOC_PATH)")
		printOpt(w, "-d, --delete", "", "Remove labels instead of adding them")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if fs.NArg() < 2 {
		return opts, errors.New("live set-label requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DbPath == "" {
		return opts, errors.New("no document source specified (use --db or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseCorpusShowArgs(args []string, ui UI) (ShowOptions, string, error) {
	fs := flag.NewFlagSet("corpus show", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ShowOptions
	fs.IntVar(&opts.Start, "start", 0, "")
	fs.IntVar(&opts.Start, "s", 0, "")

	var countOpt optionalInt
	fs.Var(&countOpt, "number", "")
	fs.Var(&countOpt, "n", "")

	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus show [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show rendered contents of a document's NLP field from the corpus staging database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --start", "INDEX", "Index of the first sentence to show (default: 0)")
		printOpt(w, "-n, --number", "N", "Number of sentences to show")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
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
		return opts, "", errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	if fs.NArg() != 1 {
		return opts, "", errors.New("corpus show requires exactly one argument: <id>")
	}
	arg := fs.Arg(0)

	return opts, arg, nil
}

func parseCorpusIngestMetaArgs(args []string, ui UI) (CorpusIngestMetaOptions, error) {
	fs := flag.NewFlagSet("corpus ingest-meta", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusIngestMetaOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.BoolVar(&opts.Pandoc, "pandoc", false, "")
	fs.BoolVar(&opts.Pandoc, "p", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ingest-meta [options] <dir>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Scan a directory for epub files and build a corpus database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "dir", "Directory to scan for epub files")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Output SQLite file for corpus data (or SEGROB_CORPUS_PATH)")
		printOpt(w, "-p, --pandoc", "", "Use pandoc for text extraction instead of pure Go")
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
		return opts, errors.New("corpus ingest-meta requires exactly one directory argument")
	}

	dir := fs.Arg(0)
	info, err := os.Stat(dir)
	if err != nil {
		return opts, fmt.Errorf("directory not found: %s", dir)
	}
	if !info.IsDir() {
		return opts, fmt.Errorf("argument is not a directory: %s", dir)
	}

	opts.Dir = dir

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	return opts, nil
}

func parseCorpusDumpTxtArgs(args []string, ui UI) (CorpusDumpTxtOptions, error) {
	fs := flag.NewFlagSet("corpus dump-txt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusDumpTxtOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.StringVar(&opts.Output, "output", "", "")
	fs.StringVar(&opts.Output, "o", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus dump-txt [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output the txt field of a corpus document byte-exact.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "-o, --output", "FILE", "Write output to FILE instead of stdout")
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
		return opts, errors.New("corpus dump-txt requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	return opts, nil
}

func parseCorpusDumpNlpArgs(args []string, ui UI) (CorpusDumpNlpOptions, error) {
	fs := flag.NewFlagSet("corpus dump-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusDumpNlpOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.BoolVar(&opts.NoLemmas, "no-lemmas", false, "")
	fs.BoolVar(&opts.NoLemmas, "n", false, "")
	fs.StringVar(&opts.Output, "output", "", "")
	fs.StringVar(&opts.Output, "o", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus dump-nlp [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output the nlp field of a corpus document.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "-n, --no-lemmas", "", "Strip lemmas from the JSON payload")
		printOpt(w, "-o, --output", "FILE", "Write output to FILE instead of stdout")
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
		return opts, errors.New("corpus dump-nlp requires exactly one argument: <id>")
	}
	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	return opts, nil
}

func parseCorpusLsArgs(args []string, ui UI) (CorpusLsOptions, error) {
	fs := flag.NewFlagSet("corpus ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusLsOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.BoolVar(&opts.WithNlp, "with-nlp", false, "")
	fs.BoolVar(&opts.WithNlp, "n", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ls [options] [filter]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all documents in the corpus staging database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "filter", "Optional substring filter on document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "-n, --with-nlp", "", "Only list records that have NLP data")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if fs.NArg() > 0 {
		opts.Filter = fs.Arg(0)
	}

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	return opts, nil
}

func setupUsage(fs *flag.FlagSet) {
	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s command [command options] [arguments...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Sentence dictionary based on NLP topics\n")

		fmt.Fprintf(w, "\nCommands: Corpus - Stage\n")
		fmt.Fprintf(w, helpCmdFmt, "corpus", "Manage the corpus staging database.")

		fmt.Fprintf(w, "\nCommands: Doc - Live - Production\n")
		fmt.Fprintf(w, helpCmdFmt, "live", "Manage the live production database.")
		fmt.Fprintf(w, helpCmdFmt, "import-topic", "Import topics from filesystem to SQLite.")
		fmt.Fprintf(w, helpCmdFmt, "export-topic", "Export topics from SQLite to filesystem.")

		fmt.Fprintf(w, "\nCommands: Other\n")
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
