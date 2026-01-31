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
	Labels   []string
	NoColor  bool
	NoPrefix bool
	Doc      *int // nil = not set
	Sent     *int // nil = not set
	NMatches int
	Format   string
	DocPath  string
}

type QueryOptions struct {
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

type ImportDocOptions struct {
	From string
	To   string
}

type ExportDocOptions struct {
	From string
	To   string
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

func parseDocArgs(args []string, ui UI) (DocOptions, int, error) {
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
			return opts, 0, err
		}
		return opts, 0, err
	}

	opts.Count = countOpt.value

	if opts.DocPath == "" {
		return opts, 0, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	arg := fs.Arg(0)
	if arg == "" {
		return opts, 0, errors.New("document ID required")
	}

	id, err := strconv.Atoi(arg)
	if err != nil {
		return opts, 0, fmt.Errorf("invalid document ID '%s': %w", arg, err)
	}

	return opts, id, nil
}

func parseLsDocArgs(args []string, ui UI) (LsDocOptions, bool, error) {
	fs := flag.NewFlagSet("ls-doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts LsDocOptions
	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "d", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

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

func parseSentenceArgs(args []string, ui UI) (SentenceOptions, int, int, error) {
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
			return opts, 0, 0, err
		}
		return opts, 0, 0, err
	}

	if opts.DocPath == "" {
		return opts, 0, 0, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	if fs.NArg() != 2 {
		return opts, 0, 0, errors.New("sentence command needs exactly two arguments: <doc_id> <sentence_id>")
	}

	docId, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return opts, 0, 0, fmt.Errorf("invalid docID '%s': %w", fs.Arg(0), err)
	}

	sentId, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		return opts, 0, 0, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
	}

	return opts, docId, sentId, nil
}

func parseTopicsArgs(args []string, ui UI) (TopicsOptions, int, int, error) {
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
			return opts, 0, 0, err
		}
		return opts, 0, 0, err
	}

	if opts.TopicPath == "" {
		return opts, 0, 0, errors.New("topic source must be specified via -t or SEGROB_TOPIC_PATH")
	}

	if opts.DocPath == "" {
		return opts, 0, 0, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	if fs.NArg() != 2 {
		return opts, 0, 0, errors.New("topics command needs exactly two arguments: <doc_id> <sentence_id>")
	}

	docId, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return opts, 0, 0, fmt.Errorf("invalid docID '%s': %w", fs.Arg(0), err)
	}

	sentId, err := strconv.Atoi(fs.Arg(1))
	if err != nil {
		return opts, 0, 0, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
	}

	if err := validatePaths(opts.DocPath, opts.TopicPath); err != nil {
		return opts, 0, 0, err
	}

	return opts, docId, sentId, nil
}

func parseExprArgs(args []string, ui UI) (ExprOptions, []string, bool, error) {
	fs := flag.NewFlagSet("expr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ExprOptions
	labels := (*stringSliceFlag)(&opts.Labels)
	fs.Var(labels, "label", "Only scan those token files that match the labels (contains)")
	fs.Var(labels, "l", "alias for -label")

	fs.BoolVar(&opts.NoColor, "no-color", false, "Show matched sentences without formatting (color)")
	fs.BoolVar(&opts.NoColor, "c", false, "alias for -no-color")

	fs.BoolVar(&opts.NoPrefix, "no-prefix", false, "Show matched sentences without prefixes with metadata")
	fs.BoolVar(&opts.NoPrefix, "x", false, "alias for -no-prefix")

	var docOpt, sentOpt optionalInt
	fs.Var(&docOpt, "doc", "Limit searched to the doc specified by this number")
	fs.Var(&docOpt, "d", "alias for -doc")

	fs.Var(&sentOpt, "sent", "Limit searched to the sentence specified by this number. Needs --doc")
	fs.Var(&sentOpt, "s", "alias for -sent")

	fs.IntVar(&opts.NMatches, "nmatches", 0, "Only show matched sentences with score greater than this number")
	fs.IntVar(&opts.NMatches, "n", 0, "alias for -nmatches")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "Show whole sentence (all), only surrounding of matched words (part) or only matches words (lemma)")
	fs.Var(formatFlag, "f", "alias for -format")

	fs.StringVar(&opts.DocPath, "doc-path", os.Getenv("SEGROB_DOC_PATH"), "Path to docs directory or SQLite file")
	fs.StringVar(&opts.DocPath, "dp", os.Getenv("SEGROB_DOC_PATH"), "alias for -doc-path")

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

	opts.Doc = docOpt.value
	opts.Sent = sentOpt.value

	if fs.NArg() < 1 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return opts, nil, false, errors.New("expr command needs at least one argument")
	}

	if opts.DocPath == "" {
		return opts, nil, false, errors.New("Doc path must be specified via -dp or SEGROB_DOC_PATH")
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

func parseStatArgs(args []string, ui UI) (StatOptions, int, *int, error) {
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
			return opts, 0, nil, err
		}
		return opts, 0, nil, err
	}

	if opts.DocPath == "" {
		return opts, 0, nil, errors.New("document source must be specified via -d or SEGROB_DOC_PATH")
	}

	if fs.NArg() < 1 {
		return opts, 0, nil, errors.New("stat command needs at least one argument: <doc_id>")
	}

	docId, err := strconv.Atoi(fs.Arg(0))
	if err != nil {
		return opts, 0, nil, fmt.Errorf("invalid docID '%s': %w", fs.Arg(0), err)
	}

	var sentId *int
	if fs.NArg() > 1 {
		v, err := strconv.Atoi(fs.Arg(1))
		if err != nil {
			return opts, 0, nil, fmt.Errorf("invalid sentenceId '%s': %w", fs.Arg(1), err)
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

func parseImportDocArgs(args []string, ui UI) (ImportDocOptions, error) {
	fs := flag.NewFlagSet("import-doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ImportDocOptions
	fs.StringVar(&opts.From, "from", "", "Source directory with JSON docs")
	fs.StringVar(&opts.To, "to", "", "Target SQLite database file")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s import-doc --from <dir> --to <sqlite_file>\n", os.Args[0])
	}

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if opts.From == "" || opts.To == "" {
		return opts, errors.New("--from and --to are required")
	}

	return opts, nil
}

func parseExportDocArgs(args []string, ui UI) (ExportDocOptions, error) {
	fs := flag.NewFlagSet("export-doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts ExportDocOptions
	fs.StringVar(&opts.From, "from", "", "Source SQLite database file")
	fs.StringVar(&opts.To, "to", "", "Target directory for JSON docs")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s export-doc --from <sqlite_file> --to <dir>\n", os.Args[0])
	}

	if err := fs.Parse(args); err != nil {
		return opts, err
	}

	if opts.From == "" || opts.To == "" {
		return opts, errors.New("--from and --to are required")
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
		_, _ = fmt.Fprintf(output, "  sentence  Show a specific sentence details.\n")
		_, _ = fmt.Fprintf(output, "  topics    Show topics for a specific sentence.\n")
		_, _ = fmt.Fprintf(output, "  expr      Evaluate a topic expression.\n")
		_, _ = fmt.Fprintf(output, "  query     Enter interactive query mode.\n")
		_, _ = fmt.Fprintf(output, "  edit      Enter interactive edit mode.\n")
		_, _ = fmt.Fprintf(output, "  topic     List topics or show expressions of a topic.\n")
		_, _ = fmt.Fprintf(output, "  stat      Show statistics for a document or sentence.\n")
		_, _ = fmt.Fprintf(output, "  import-topic  Import topics from filesystem to SQLite.\n")
		_, _ = fmt.Fprintf(output, "  export-topic  Export topics from SQLite to filesystem.\n")
		_, _ = fmt.Fprintf(output, "  import-doc    Import docs from filesystem to SQLite.\n")
		_, _ = fmt.Fprintf(output, "  export-doc    Export docs from SQLite to filesystem.\n")
		_, _ = fmt.Fprintf(output, "  bash      Output bash completion script.\n")
		_, _ = fmt.Fprintf(output, "  help      Show help for a command.\n")
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
