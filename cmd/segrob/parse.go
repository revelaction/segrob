package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
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
}

type QueryOptions struct {
	NoColor  bool
	NoPrefix bool
	NMatches int
	Format   string
}

type TopicsOptions struct {
	Format    string
	TopicPath string
}

type DocOptions struct {
	Start int
	Count int
}

type EditOptions struct {
	TopicPath string
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

func parseDocArgs(args []string, ui UI) (DocOptions, string, bool, error) {
	fs := flag.NewFlagSet("doc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts DocOptions
	fs.IntVar(&opts.Start, "start", 0, "Index of the first sentence to show")
	fs.IntVar(&opts.Count, "n", -1, "Number of sentences to show (-1 for all)")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s doc [options] [file_path|db_id]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show contents of a document file or DB entry.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
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

	if fs.NArg() > 1 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return opts, "", false, errors.New("doc command accepts at most one argument")
	}

	if fs.NArg() == 0 {
		return opts, "", false, nil
	}

	arg := fs.Arg(0)
	isFile := false

	// Validation
	if info, err := os.Stat(arg); err == nil && !info.IsDir() {
		isFile = true
	} else {
		// regex check for digits if not a file
		digitRegex := regexp.MustCompile(`^\d+$`)
		if !digitRegex.MatchString(arg) {
			return opts, "", false, fmt.Errorf("file not found and not a valid DB ID: %s", arg)
		}
	}

	return opts, arg, isFile, nil
}

func parseSentenceArgs(args []string, ui UI) (string, int, bool, error) {
	fs := flag.NewFlagSet("sentence", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s sentence <source> <sentenceId>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show a specific sentence details. <source> can be a file path or a DB ID.\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return "", 0, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return "", 0, false, err
	}

	if fs.NArg() != 2 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return "", 0, false, errors.New("sentence command needs exactly two arguments: <source> <sentenceId>")
	}

	source := fs.Arg(0)
	sentId, sentErr := strconv.Atoi(fs.Arg(1))
	if sentErr != nil {
		return "", 0, false, fmt.Errorf("invalid sentenceId: %v", sentErr)
	}

	isFile := false
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		isFile = true
	} else {
		// regex check for digits if not a file
		digitRegex := regexp.MustCompile(`^\d+$`)
		if !digitRegex.MatchString(source) {
			return "", 0, false, fmt.Errorf("source not found and not a valid DB ID: %s", source)
		}
	}

	return source, sentId, isFile, nil
}

func parseTopicsArgs(args []string, ui UI) (TopicsOptions, string, int, bool, error) {
	fs := flag.NewFlagSet("topics", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts TopicsOptions
	fs.StringVar(&opts.TopicPath, "topics-dir", os.Getenv("SEGROB_TOPIC_PATH"), "Path to topics directory")
	fs.StringVar(&opts.TopicPath, "t", os.Getenv("SEGROB_TOPIC_PATH"), "alias for -topics-dir")

	opts.Format = render.Defaultformat
	formatFlag := &enumFlag{allowed: render.SupportedFormats(), value: &opts.Format}
	fs.Var(formatFlag, "format", "Show whole sentence (all), only surrounding of matched words (part) or only matches words (lemma)")
	fs.Var(formatFlag, "f", "alias for -format")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s topics [options] <source> <sentenceId>\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show topics for a specific sentence. <source> can be a file path or a DB ID.\n")
		_, _ = fmt.Fprintf(fs.Output(), "\nOptions:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, "", 0, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, "", 0, false, err
	}

	if opts.TopicPath == "" {
		return opts, "", 0, false, errors.New("Topics path must be specified via --topics-dir or SEGROB_TOPIC_PATH")
	}

	if info, err := os.Stat(opts.TopicPath); err != nil || !info.IsDir() {
		return opts, "", 0, false, fmt.Errorf("Topics directory not found or not a directory: %s", opts.TopicPath)
	}

	if fs.NArg() != 2 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return opts, "", 0, false, errors.New("topics command needs two arguments: <source> <sentenceId>")
	}

	source := fs.Arg(0)
	sentId, sentErr := strconv.Atoi(fs.Arg(1))
	if sentErr != nil {
		return opts, "", 0, false, fmt.Errorf("invalid sentenceId: %v", sentErr)
	}

	isFile := false
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		isFile = true
	} else {
		// regex check for digits if not a file
		digitRegex := regexp.MustCompile(`^\d+$`)
		if !digitRegex.MatchString(source) {
			return opts, "", 0, false, fmt.Errorf("source not found and not a valid DB ID: %s", source)
		}
	}

	return opts, source, sentId, isFile, nil
}

func parseExprArgs(args []string, ui UI) (ExprOptions, []string, error) {
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
			return opts, nil, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, nil, err
	}

	opts.Doc = docOpt.value
	opts.Sent = sentOpt.value

	if fs.NArg() < 1 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return opts, nil, errors.New("expr command needs at least one argument")
	}

	return opts, fs.Args(), nil
}

func parseQueryArgs(args []string, ui UI) (QueryOptions, error) {
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

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s query [options]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Enter interactive query mode.\n")
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
	return opts, nil
}

func parseEditArgs(args []string, ui UI) (EditOptions, error) {
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
			return opts, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return opts, err
	}

	if opts.TopicPath == "" {
		return opts, errors.New("Topic path must be specified via -t or SEGROB_TOPIC_PATH")
	}

	if _, err := os.Stat(opts.TopicPath); err != nil {
		return opts, fmt.Errorf("Topic path not found: %s", opts.TopicPath)
	}

	return opts, nil
}

func parseTopicArgs(args []string, ui UI) (string, error) {
	fs := flag.NewFlagSet("topic", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s topic [name]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  List topics or show expressions of a topic.\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return "", err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return "", err
	}

	if fs.NArg() > 0 {
		return fs.Arg(0), nil
	}
	return "", nil
}

func parseStatArgs(args []string, ui UI) (string, *int, bool, error) {
	fs := flag.NewFlagSet("stat", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "Usage: %s stat <source> [sentenceId]\n", os.Args[0])
		_, _ = fmt.Fprintf(fs.Output(), "\nDescription:\n")
		_, _ = fmt.Fprintf(fs.Output(), "  Show statistics for a document or sentence. <source> can be a file path or a DB ID.\n")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return "", nil, false, err
		}
		fs.SetOutput(ui.Err)
		fprintErr(ui.Err, err)
		fs.Usage()
		return "", nil, false, err
	}

	if fs.NArg() == 0 {
		fs.SetOutput(ui.Err)
		fs.Usage()
		return "", nil, false, errors.New("stat command needs at least one argument")
	}

	source := fs.Arg(0)
	var sentId *int
	if fs.NArg() > 1 {
		v, err := strconv.Atoi(fs.Arg(1))
		if err != nil {
			return "", nil, false, fmt.Errorf("invalid sentenceId: %v", err)
		}
		sentId = &v
	}

	isFile := false
	if info, err := os.Stat(source); err == nil && !info.IsDir() {
		isFile = true
	} else {
		// regex check for digits if not a file
		digitRegex := regexp.MustCompile(`^\d+$`)
		if !digitRegex.MatchString(source) {
			return "", nil, false, fmt.Errorf("source not found and not a valid DB ID: %s", source)
		}
	}

	return source, sentId, isFile, nil
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

func setupUsage(fs *flag.FlagSet) {
	fs.Usage = func() {
		output := fs.Output()
		_, _ = fmt.Fprintf(output, "Usage: %s command [command options] [arguments...]\n", os.Args[0])
		_, _ = fmt.Fprintf(output, "\nDescription:\n")
		_, _ = fmt.Fprintf(output, "  Sentence dictionary based on NLP topics\n")
		_, _ = fmt.Fprintf(output, "\nCommands:\n")
		_, _ = fmt.Fprintf(output, "  doc       List metadata of all/some token files.\n")
		_, _ = fmt.Fprintf(output, "  sentence  Show a specific sentence details.\n")
		_, _ = fmt.Fprintf(output, "  topics    Show topics for a specific sentence.\n")
		_, _ = fmt.Fprintf(output, "  expr      Evaluate a topic expression.\n")
		_, _ = fmt.Fprintf(output, "  query     Enter interactive query mode.\n")
		_, _ = fmt.Fprintf(output, "  edit      Enter interactive edit mode.\n")
		_, _ = fmt.Fprintf(output, "  topic     List topics or show expressions of a topic.\n")
		_, _ = fmt.Fprintf(output, "  stat      Show statistics for a document or sentence.\n")
		_, _ = fmt.Fprintf(output, "  bash      Output bash completion script.\n")
		_, _ = fmt.Fprintf(output, "  help      Show help for a command.\n")
	}
}
