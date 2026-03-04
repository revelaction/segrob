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

type LsDocOptions struct {
	DocPath string
	Match   string
}

type LsLabelsOptions struct {
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
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s live <id> [--from corpus.db] [--to segrob.db] [-m] [--keep-nlp]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Move a document from corpus staging to live production tables.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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

type AddLabelOptions struct {
	DocID   string
	Labels  []string
	DocPath string
}

type RemoveLabelOptions struct {
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

type CorpusLsOptions struct {
	DbPath string // --db / SEGROB_CORPUS_PATH
	Filter string // optional positional filter
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
	fs.IntVar(&opts.Start, "start", 0, "Index of the first sentence to show")
	fs.IntVar(&opts.Start, "s", 0, "alias for -start")

	var countOpt optionalInt
	fs.Var(&countOpt, "number", "Number of sentences to show")
	fs.Var(&countOpt, "n", "alias for -number")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s doc [options] <doc_id>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show contents of a document from the configured repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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

func parseLsDocArgs(args []string, ui UI) (LsDocOptions, bool, error) {
	fs := flag.NewFlagSet("ls-doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LsDocOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")
	fs.StringVar(&opts.Match, "match", "", "Retrieve only documents with at least one label containing this string")
	fs.StringVar(&opts.Match, "m", "", "alias for -match")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s ls-doc [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  List all documents in the repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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

func parseLsLabelsArgs(args []string, ui UI) (LsLabelsOptions, error) {
	fs := flag.NewFlagSet("ls-labels", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LsLabelsOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")
	fs.StringVar(&opts.Match, "match", "", "Retrieve only labels containing this string")
	fs.StringVar(&opts.Match, "m", "", "alias for -match")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s ls-labels [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  List all unique labels in the repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s sentence [options] <doc_id> <sentence_id>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show a specific sentence details from the configured repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory or SQLite file")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topic-path")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "Show whole sentence (all), only surrounding of matched words (part) or only matches words (lemma)")
	fs.Var(formatFlag, "f", "alias for -format")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s topics [options] <doc_id> <sentence_id>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show topics for a specific sentence from the configured repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
	fs.Var(labels, "label", "Only scan those token files that match the labels (EXACT match, ALL labels required)")
	fs.Var(labels, "l", "alias for -label")

	fs.BoolVar(&opts.NoColor, "no-color", false, "Show matched sentences without formatting (color)")
	fs.BoolVar(&opts.NoColor, "c", false, "alias for -no-color")

	fs.BoolVar(&opts.NoPrefix, "no-prefix", false, "Show matched sentences without prefixes with metadata")
	fs.BoolVar(&opts.NoPrefix, "x", false, "alias for -no-prefix")

	fs.IntVar(&opts.NMatches, "nmatches", 0, "Only show matched sentences with score greater than this number")
	fs.IntVar(&opts.NMatches, "n", 0, "alias for -nmatches")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "Show whole sentence (all), only surrounding of matched words (part) or only matches words (lemma)")
	fs.Var(formatFlag, "f", "alias for -format")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory or SQLite file")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topic-path")

	fs.BoolVar(&opts.JSON, "json", false, "Output results as JSON")
	fs.BoolVar(&opts.JSON, "j", false, "alias for -json")

	fs.IntVar(&opts.Limit, "limit", 0, "Maximum number of matched results to return (0 = unlimited)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s expr [options] <topic expr item> ...\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Evaluate a topic expression.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
	fs.Var(labels, "label", "Only scan those token files that match the labels (EXACT match, ALL labels required)")
	fs.Var(labels, "l", "alias for -label")

	fs.BoolVar(&opts.NoColor, "no-color", false, "Show matched sentences without formatting (color)")
	fs.BoolVar(&opts.NoColor, "c", false, "alias for -no-color")

	fs.BoolVar(&opts.NoPrefix, "no-prefix", false, "Show matched sentences without prefixes with metadata")
	fs.BoolVar(&opts.NoPrefix, "x", false, "alias for -no-prefix")

	fs.IntVar(&opts.NMatches, "nmatches", 0, "Only show matched sentences with score greater than this number")
	fs.IntVar(&opts.NMatches, "n", 0, "alias for -nmatches")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "Show whole sentence (all), only surrounding of matched words (part) or only matches words (lemma)")
	fs.Var(formatFlag, "f", "alias for -format")

	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory or SQLite file")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topic-path")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s query [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Enter interactive query mode.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
		_, _ = fmt.Fprintf(fs.Output(), "  -t, --topic-path    Path to topics directory or SQLite file (required, or set SEGROB_TOPIC_PATH)\n")
		_, _ = fmt.Fprintf(fs.Output(), "  -d, --doc-path      Path to docs directory or SQLite file (required, or set SEGROB_DOC_PATH)\n")
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
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory or SQLite file")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topic-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s edit [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Enter interactive edit mode.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  -t, --topic-path    Path to topics directory or SQLite file (required, or set SEGROB_TOPIC_PATH)\n")
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
	fs.StringVar(&opts.TopicPath, "topic-path", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory or SQLite file")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topic-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s topic [options] [name]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  List topics or show expressions of a topic.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  -t, --topic-path    Path to topics directory or SQLite file (required, or set SEGROB_TOPIC_PATH)\n")
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
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s stat [options] <doc_id> [sentence_id]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show statistics for a document or sentence from the configured repository.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s bash\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Output bash completion script.\n")
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
	fs.StringVar(&opts.From, "from", "", "Source directory with JSON topics")
	fs.StringVar(&opts.To, "to", "", "Target SQLite database file")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s import-topic --from <dir> --to <sqlite_file>\n", os.Args[0])
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
	fs.StringVar(&opts.From, "from", "", "Source SQLite database file")
	fs.StringVar(&opts.To, "to", "", "Target directory for JSON topics")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s export-topic --from <sqlite_file> --to <dir>\n", os.Args[0])
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
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s init-db <db>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Initialize a new SQLite database with the required schema.\n")
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
	var opts CorpusNlpOptions
	fs.StringVar(&opts.NlpScript, "nlp-script", os.Getenv("SEGROB_NLP_SCRIPT"), "path to python NLP script")
	fs.StringVar(&opts.NlpScript, "s", os.Getenv("SEGROB_NLP_SCRIPT"), "path to python NLP script (shorthand)")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "path to corpus database")

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

func parseAddLabelArgs(args []string, ui UI) (AddLabelOptions, error) {
	fs := flag.NewFlagSet("add-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts AddLabelOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s add-label [options] <doc_id> <label> [<label>...]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Add one or more labels to a document.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
		return opts, errors.New("add-label requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DocPath == "" {
		return opts, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseRemoveLabelArgs(args []string, ui UI) (RemoveLabelOptions, error) {
	fs := flag.NewFlagSet("remove-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts RemoveLabelOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s remove-label [options] <doc_id> <label> [<label>...]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Remove one or more labels from a document.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
		return opts, errors.New("remove-label requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DocPath == "" {
		return opts, errors.New("no document source specified (use -d or SEGROB_DOC_PATH)")
	}

	return opts, nil
}

func parseCorpusMetaArgs(args []string, ui UI) (CorpusMetaOptions, error) {
	fs := flag.NewFlagSet("corpus-meta", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusMetaOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "Output SQLite file for corpus data")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s corpus-meta [options] <dir>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Scan a directory for epub files and build a corpus database.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "Corpus SQLite file")
	fs.StringVar(&opts.Output, "output", "", "Output file path (default: stdout)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s cat-txt <id> [--db corpus.db] [--output file.txt]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Output the txt field of a corpus document byte-exact.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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

func parseCorpusLsArgs(args []string, ui UI) (CorpusLsOptions, error) {
	fs := flag.NewFlagSet("corpus-ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusLsOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_PATH"), "Corpus SQLite file")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s corpus-ls [filter] [--db corpus.db]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  List all documents in the corpus staging database.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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
		output := fs.Output()
		_, _ = fmt.Fprintf(output, "Usage: %s command [command options] [arguments...]\n", os.Args[0])
		_, _ = fmt.Fprintf(output, "\nDescription:\n")
		_, _ = fmt.Fprintf(output, "  Sentence dictionary based on NLP topics\n")
		_, _ = fmt.Fprintf(output, "\nCommands:\n")
		_, _ = fmt.Fprintf(output, "  doc       Show contents of a document file or DB entry.\n")
		_, _ = fmt.Fprintf(output, "  ls-doc    List all documents in the repository.\n")
		_, _ = fmt.Fprintf(output, "  ls-labels List all unique labels in the repository.\n")
		_, _ = fmt.Fprintf(output, "  sentence  Show a specific sentence details.\n")
		_, _ = fmt.Fprintf(output, "  topics    Show topics for a specific sentence.\n")
		_, _ = fmt.Fprintf(output, "  expr      Evaluate a topic expression.\n")
		_, _ = fmt.Fprintf(output, "  query     Enter interactive query mode.\n")
		_, _ = fmt.Fprintf(output, "  edit      Enter interactive edit mode.\n")
		_, _ = fmt.Fprintf(output, "  topic     List topics or show expressions of a topic.\n")
		_, _ = fmt.Fprintf(output, "  stat      Show statistics for a document or sentence.\n")
		_, _ = fmt.Fprintf(output, "  import-topic  Import topics from filesystem to SQLite.\n")
		_, _ = fmt.Fprintf(output, "  export-topic  Export topics from SQLite to filesystem.\n")
		_, _ = fmt.Fprintf(output, "  init-db       Initialize a new SQLite database with the required schema\n")
		_, _ = fmt.Fprintf(output, "  corpus-meta   Scan an epub directory and build a corpus database.\n")
		_, _ = fmt.Fprintf(output, "  corpus-nlp    Process document text with NLP and store in corpus.\n")
		_, _ = fmt.Fprintf(output, "  corpus-ls     List documents in the corpus staging database.\n")
		_, _ = fmt.Fprintf(output, "  live          Move a document from corpus to live production tables.\n")
		_, _ = fmt.Fprintf(output, "  add-label     Add one or more labels to a document.\n")
		_, _ = fmt.Fprintf(output, "  remove-label  Remove one or more labels from a document.\n")
		_, _ = fmt.Fprintf(output, "  cat-txt       Output the txt field of a corpus document.\n")
		_, _ = fmt.Fprintf(output, "  bash          Output bash completion script.\n")
		_, _ = fmt.Fprintf(output, "  version   Show version information\n")
		_, _ = fmt.Fprintf(output, "  help      Show help for a command.\n")
		_, _ = fmt.Fprintf(output, "\nVersion: %s, commit %s\n", BuildTag, BuildCommit)
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
