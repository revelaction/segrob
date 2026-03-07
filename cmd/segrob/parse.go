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
type ExprOptions struct {
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

type QueryOptions struct {
	Labels    []string
	NoColor   bool
	NoPrefix  bool
	NMatches  int
	Format    string
	TopicPath string
	DocPath   string
}

type TopicsOptions struct {
	Format    string
	TopicPath string
	DocPath   string
}

type TopicOptions struct {
	TopicPath string
}

type DocOptions struct {
	Start   int
	Count   *int
	DocPath string
}

type DocLsOptions struct {
	DocPath string
	Match   string
}

type LabelLsOptions struct {
	DocPath string
	Match   string
}

type SentenceOptions struct {
	DocPath string
}

type StatOptions struct {
	DocPath string
}

type EditOptions struct {
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

type InitDbOptions struct {
	DbPath string
}

type CorpusNlpOptions struct {
	NlpScript string
	DbPath    string // corpus db path
	ID        string
}

type LiveOptions struct {
	From string // corpus.db path (--from / SEGROB_CORPUS_PATH)
	To   string // segrob.db path (--to / SEGROB_DOC_PATH)
	ID   string // positional arg: document id
	Move bool   // -m/--move: delete nlp from corpus after success
}

func parseLiveArgs(args []string, ui UI) (LiveOptions, error) {
	fs := flag.NewFlagSet("live", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LiveOptions
	fs.StringVar(&opts.From, "from", os.Getenv("SEGROB_CORPUS_PATH"), "Source corpus SQLite file")
	fs.StringVar(&opts.To, "to", os.Getenv("SEGROB_DOC_PATH"), "Target segrob SQLite file")
	fs.BoolVar(&opts.Move, "move", false, "Delete nlp data from corpus after successful live")
	fs.BoolVar(&opts.Move, "m", false, "alias for -move")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s live [options] <id>\n\n", os.Args[0])
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
		return opts, errors.New("live requires exactly one argument: <id>")
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

type LabelAddOptions struct {
	DocID   string
	Labels  []string
	DocPath string
}

type LabelRmOptions struct {
	DocID   string
	Labels  []string
	DocPath string
}

type CorpusMetaOptions struct {
	DbPath string
	Dir    string
}

type CatTxtOptions struct {
	DbPath string // --db / SEGROB_CORPUS_PATH
	Output string // --output file path (empty = stdout)
	ID     string // positional arg: document id
}

type CatNlpOptions struct {
	DbPath   string // --db / SEGROB_CORPUS_PATH
	NoLemmas bool   // -n, --no-lemmas
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

func parseDocArgs(args []string, ui UI) (DocOptions, string, error) {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts DocOptions
	fs.IntVar(&opts.Start, "start", 0, "")
	fs.IntVar(&opts.Start, "s", 0, "")

	var countOpt optionalInt
	fs.Var(&countOpt, "number", "")
	fs.Var(&countOpt, "n", "")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s doc [options] <doc_id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show contents of a document from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document to show")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --start", "INDEX", "Index of the first sentence to show (default: 0)")
		printOpt(w, "-n, --number", "N", "Number of sentences to show")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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

	if opts.DocPath == "" {
		return opts, "", errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	arg := fs.Arg(0)
	if arg == "" {
		return opts, "", errors.New("document ID required")
	}

	return opts, arg, nil
}

func parseDocLsArgs(args []string, ui UI) (DocLsOptions, bool, error) {
	fs := flag.NewFlagSet("doc-ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts DocLsOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s doc-ls [options]\n\n", os.Args[0])
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

func parseLabelLsArgs(args []string, ui UI) (LabelLsOptions, error) {
	fs := flag.NewFlagSet("label-ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LabelLsOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s label-ls [options]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all unique labels in the repository.\n")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
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

	if opts.DocPath == "" {
		return opts, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseSentenceArgs(args []string, ui UI) (SentenceOptions, string, int, error) {
	fs := flag.NewFlagSet("sentence", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts SentenceOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s sentence [options] <doc_id> <sentence_id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show details of a specific sentence from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "sentence_id", "Index of the sentence")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", 0, err
		}
		return opts, "", 0, err
	}

	if opts.DocPath == "" {
		return opts, "", 0, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	if fs.NArg() != 2 {
		return opts, "", 0, errors.New("sentence command needs exactly two arguments: <doc_id> <sentence_id>")
	}

	docId := fs.Arg(0)

	sentId, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		return opts, "", 0, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
	}

	return opts, docId, sentId, nil
}

func parseTopicsArgs(args []string, ui UI) (TopicsOptions, string, int, error) {
	fs := flag.NewFlagSet("topics", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts TopicsOptions
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
		fmt.Fprintf(w, "Usage: %s topics [options] <doc_id> <sentence_id>\n\n", os.Args[0])
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
		return opts, "", 0, errors.New("topics command needs exactly two arguments: <doc_id> <sentence_id>")
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

func parseExprArgs(args []string, ui UI) (ExprOptions, []string, bool, error) {
	fs := flag.NewFlagSet("expr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ExprOptions
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
		fmt.Fprintf(w, "Usage: %s expr [options] <expr>...\n\n", os.Args[0])
		fmt.Fprintf(w, "  Evaluate a topic expression.\n")
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
		return opts, nil, false, errors.New("expr command needs at least one argument")
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

func parseQueryArgs(args []string, ui UI) (QueryOptions, bool, bool, error) {
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts QueryOptions
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
		fmt.Fprintf(w, "Usage: %s query [options]\n\n", os.Args[0])
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

func parseEditArgs(args []string, ui UI) (EditOptions, bool, error) {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts EditOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s edit [options]\n\n", os.Args[0])
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

func parseTopicArgs(args []string, ui UI) (TopicOptions, string, bool, error) {
	fs := flag.NewFlagSet("topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts TopicOptions
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s topic [options] [name]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List topics or show expressions of a named topic.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "name", "Topic name to inspect (optional; lists all topics if omitted)")
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

	info, err := os.Stat(opts.TopicPath)
	if err != nil {
		return opts, "", false, fmt.Errorf("Topic path not found: %s", opts.TopicPath)
	}

	name := ""
	if fs.NArg() > 0 {
		name = fs.Arg(0)
	}

	return opts, name, !info.IsDir(), nil
}

func parseStatArgs(args []string, ui UI) (StatOptions, string, *int, error) {
	fs := flag.NewFlagSet("stat", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts StatOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s stat [options] <doc_id> [sentence_id]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show statistics for a document or sentence from the configured repository.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "sentence_id", "Index of the sentence (optional)")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to docs directory or SQLite file (or SEGROB_DOC_PATH)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", nil, err
		}
		return opts, "", nil, err
	}

	if opts.DocPath == "" {
		return opts, "", nil, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	if fs.NArg() < 1 {
		return opts, "", nil, errors.New("stat command needs at least one argument: <doc_id>")
	}

	docId := fs.Arg(0)

	var sentId *int
	if fs.NArg() > 1 {
		v, err := strconv.Atoi(fs.Arg(1))
		if err != nil {
			return opts, "", nil, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
		}
		sentId = &v
	}

	return opts, docId, sentId, nil
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

func parseInitDbArgs(args []string, ui UI) (InitDbOptions, error) {
	fs := flag.NewFlagSet("init-db", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts InitDbOptions

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s init-db <db>\n\n", os.Args[0])
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
		return opts, errors.New("init-db requires exactly one argument: <db>")
	}

	opts.DbPath = fs.Arg(0)
	return opts, nil
}

func parseCorpusNlp(args []string) (CorpusNlpOptions, error) {
	fs := flag.NewFlagSet("corpus-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusNlpOptions
	fs.StringVar(&opts.NlpScript, "nlp-script", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.NlpScript, "s", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus-nlp [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Process document text with NLP and store results in the corpus.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to process")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --nlp-script", "PATH", "Path to the Python NLP script (or SEGROB_NLP_SCRIPT)")
		printOpt(w, "--db", "FILE", "Path to the corpus SQLite database (or SEGROB_CORPUS_PATH)")
	}

	if err := fs.Parse(args); err != nil {
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

func parseLabelAddArgs(args []string, ui UI) (LabelAddOptions, error) {
	fs := flag.NewFlagSet("label-add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LabelAddOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s label-add [options] <doc_id> <label> [<label>...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Add one or more labels to a document.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "label", "One or more labels to add")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, errors.New("label-add requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DocPath == "" {
		return opts, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseLabelRmArgs(args []string, ui UI) (LabelRmOptions, error) {
	fs := flag.NewFlagSet("label-rm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LabelRmOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s label-rm [options] <doc_id> <label> [<label>...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Remove one or more labels from a document.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "label", "One or more labels to remove")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-d, --doc-path", "PATH", "Path to SQLite file (or SEGROB_DOC_PATH)")
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
		return opts, errors.New("label-rm requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DocPath == "" {
		return opts, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

type CorpusDocOptions struct {
	Start  int
	Count  *int
	DbPath string
}

func parseCorpusDocArgs(args []string, ui UI) (CorpusDocOptions, string, error) {
	fs := flag.NewFlagSet("corpus-doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusDocOptions
	fs.IntVar(&opts.Start, "start", 0, "")
	fs.IntVar(&opts.Start, "s", 0, "")

	var countOpt optionalInt
	fs.Var(&countOpt, "number", "")
	fs.Var(&countOpt, "n", "")

	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus-doc [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show contents of a document's NLP field from the corpus staging database.\n")
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
		return opts, "", errors.New("corpus-doc requires exactly one argument: <id>")
	}
	arg := fs.Arg(0)

	return opts, arg, nil
}

func parseCorpusMetaArgs(args []string, ui UI) (CorpusMetaOptions, error) {
	fs := flag.NewFlagSet("corpus-meta", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusMetaOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus-meta [options] <dir>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Scan a directory for epub files and build a corpus database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "dir", "Directory to scan for epub files")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Output SQLite file for corpus data (or SEGROB_CORPUS_PATH)")
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
		return opts, errors.New("corpus-meta requires exactly one directory argument")
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

func parseCatTxtArgs(args []string, ui UI) (CatTxtOptions, error) {
	fs := flag.NewFlagSet("cat-txt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CatTxtOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.StringVar(&opts.Output, "output", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s cat-txt [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output the txt field of a corpus document byte-exact.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "--output", "FILE", "Write output to FILE instead of stdout")
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
		return opts, errors.New("cat-txt requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_PATH")
	}

	return opts, nil
}

func parseCatNlpArgs(args []string, ui UI) (CatNlpOptions, error) {
	fs := flag.NewFlagSet("cat-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CatNlpOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.BoolVar(&opts.NoLemmas, "no-lemmas", false, "")
	fs.BoolVar(&opts.NoLemmas, "n", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s cat-nlp [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output the nlp field of a corpus document.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_PATH)")
		printOpt(w, "-n, --no-lemmas", "", "Strip lemmas from the JSON payload")
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
		return opts, errors.New("cat-nlp requires exactly one argument: <id>")
	}
	opts.ID = fs.Arg(0)

	return opts, nil
}

func parseCorpusLsArgs(args []string, ui UI) (CorpusLsOptions, error) {
	fs := flag.NewFlagSet("corpus-ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusLsOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "")
	fs.BoolVar(&opts.WithNlp, "with-nlp", false, "")
	fs.BoolVar(&opts.WithNlp, "n", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus-ls [options] [filter]\n\n", os.Args[0])
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
		fmt.Fprintf(w, helpCmdFmt, "corpus-meta", "Scan an epub directory and build a corpus database.")
		fmt.Fprintf(w, helpCmdFmt, "corpus-nlp", "Process document text with NLP and store in corpus.")
		fmt.Fprintf(w, helpCmdFmt, "corpus-ls", "List documents in the corpus staging database.")
		fmt.Fprintf(w, helpCmdFmt, "cat-txt", "Output the txt field of a corpus document.")
		fmt.Fprintf(w, helpCmdFmt, "cat-nlp", "Output the nlp field of a corpus document.")
		fmt.Fprintf(w, helpCmdFmt, "corpus-doc", "Show rendered contents of a corpus document's NLP field.")

		fmt.Fprintf(w, "\nCommands: Doc - Live - Production\n")
		fmt.Fprintf(w, helpCmdFmt, "live", "Move a document from corpus to live production tables.")
		fmt.Fprintf(w, helpCmdFmt, "doc", "Show contents of a document file or DB entry.")
		fmt.Fprintf(w, helpCmdFmt, "doc-ls", "List all documents in the repository.")
		fmt.Fprintf(w, helpCmdFmt, "label-ls", "List all unique labels in the repository.")
		fmt.Fprintf(w, helpCmdFmt, "label-add", "Add one or more labels to a document.")
		fmt.Fprintf(w, helpCmdFmt, "label-rm", "Remove one or more labels from a document.")
		fmt.Fprintf(w, helpCmdFmt, "sentence", "Show a specific sentence details.")
		fmt.Fprintf(w, helpCmdFmt, "topics", "Show topics for a specific sentence.")
		fmt.Fprintf(w, helpCmdFmt, "expr", "Evaluate a topic expression.")
		fmt.Fprintf(w, helpCmdFmt, "query", "Enter interactive query mode.")
		fmt.Fprintf(w, helpCmdFmt, "edit", "Enter interactive edit mode.")
		fmt.Fprintf(w, helpCmdFmt, "topic", "List topics or show expressions of a topic.")
		fmt.Fprintf(w, helpCmdFmt, "stat", "Show statistics for a document or sentence.")
		fmt.Fprintf(w, helpCmdFmt, "import-topic", "Import topics from filesystem to SQLite.")
		fmt.Fprintf(w, helpCmdFmt, "export-topic", "Export topics from SQLite to filesystem.")
		fmt.Fprintf(w, helpCmdFmt, "init-db", "Initialize a new SQLite database with the required schema.")

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
