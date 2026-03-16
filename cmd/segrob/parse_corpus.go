package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

type CorpusAckOptions struct {
	Nlp    bool
	By     string
	ID     string
	DbPath string
}

type CorpusRmOptions struct {
	ID     string
	DbPath string
}

type CorpusPushTxtOptions struct {
	By     string
	Note   string
	ID     string
	File   string
	DbPath string
}

type CorpusIngestNlpOptions struct {
	NlpScript string
	DbPath    string // corpus db path
	ID        string
	Force     bool // -f/--force
}

type CorpusPublishOptions struct {
	From  string // corpus.db path (--from / SEGROB_CORPUS_DB)
	To    string // segrob.db path (--to / SEGROB_LIVE_DB)
	ID    string // positional arg: document id (empty when All is true)
	All   bool   // true when no positional arg → publish all ACKed
	Move  bool   // -m/--move: delete nlp from corpus after success
	Force bool   // -f/--force
}

type CorpusPublishLabelOptions struct {
	From string // --from / SEGROB_CORPUS_DB
	To   string // --to   / SEGROB_LIVE_DB
	ID   string // positional arg: document id
}

func parseCorpusPublishArgs(args []string, ui UI) (CorpusPublishOptions, error) {
	fs := flag.NewFlagSet("corpus publish", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusPublishOptions
	fs.StringVar(&opts.From, "from", os.Getenv("SEGROB_CORPUS_DB"), "Source corpus SQLite file")
	fs.StringVar(&opts.To, "to", os.Getenv("SEGROB_LIVE_DB"), "Target segrob SQLite file")
	fs.BoolVar(&opts.Move, "move", false, "Delete nlp data from corpus after successful live")
	fs.BoolVar(&opts.Move, "m", false, "alias for -move")
	fs.BoolVar(&opts.Force, "force", false, "Force publishing even if not acknowledged")
	fs.BoolVar(&opts.Force, "f", false, "alias for -force")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus publish [options] [id]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Move documents from corpus staging to live production tables.\n")
		fmt.Fprintf(w, "  When no id is given, publishes all acknowledged documents.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to publish (omit to publish all ACKed)")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--from", "PATH", "Source corpus SQLite file (or SEGROB_CORPUS_DB)")
		printOpt(w, "--to", "PATH", "Target segrob SQLite file (or SEGROB_LIVE_DB)")
		printOpt(w, "-m, --move", "", "Delete NLP data from corpus after successful publish")
		printOpt(w, "-f, --force", "", "Force publishing even if not acknowledged (only with id)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	switch fs.NArg() {
	case 0:
		opts.All = true
	case 1:
		opts.ID = fs.Arg(0)
	default:
		return opts, errors.New("corpus publish accepts zero or one argument: [id]")
	}

	if opts.From == "" {
		return opts, errors.New("corpus source must be specified via --from or SEGROB_CORPUS_DB")
	}
	if opts.To == "" {
		return opts, errors.New("target db must be specified via --to or SEGROB_LIVE_DB")
	}

	return opts, nil
}

func parseCorpusPublishLabelArgs(args []string, ui UI) (CorpusPublishLabelOptions, error) {
	fs := flag.NewFlagSet("corpus publish-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusPublishLabelOptions
	fs.StringVar(&opts.From, "from", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.StringVar(&opts.To, "to", os.Getenv("SEGROB_LIVE_DB"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus publish-label [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Push the current corpus labels for <id> into the live tables.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--from", "PATH", "Source corpus SQLite file (or SEGROB_CORPUS_DB)")
		printOpt(w, "--to", "PATH", "Target segrob SQLite file (or SEGROB_LIVE_DB)")
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
		return opts, errors.New("corpus publish-label requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.From == "" {
		return opts, errors.New("corpus source must be specified via --from or SEGROB_CORPUS_DB")
	}
	if opts.To == "" {
		return opts, errors.New("target db must be specified via --to or SEGROB_LIVE_DB")
	}

	return opts, nil
}

type CorpusIngestMetaOptions struct {
	DbPath string
	Dir    string
	Pandoc bool
}

type CorpusDumpTxtOptions struct {
	DbPath string // --db / SEGROB_CORPUS_DB
	Output string // --output file path (empty = stdout)
	ID     string // positional arg: document id
}

type CorpusDumpNlpOptions struct {
	DbPath   string // --db / SEGROB_CORPUS_DB
	NoLemmas bool   // -n, --no-lemmas
	Output   string // --output file path (empty = stdout)
	ID       string // positional arg: document id
}

type CorpusLsOptions struct {
	DbPath  string // --db / SEGROB_CORPUS_DB
	Filter  string // optional positional filter
	WithNlp bool   // --with-nlp / -w
	NlpAck  bool   // --nlp-ack / -n
	TxtAck  bool   // --txt-ack / -t
	Ack     bool   // --ack / -a
}

// CorpusSetLabelOptions holds options for "corpus set-label".
type CorpusSetLabelOptions struct {
	DocID  string
	Labels []string
	DbPath string
	Delete bool
}

// CorpusLsLabelOptions holds options for "corpus ls-label".
type CorpusLsLabelOptions struct {
	DbPath string
	Match  string
	ID     string // Optional document ID
}

func parseCorpusIngestNlpArgs(args []string, ui UI) (CorpusIngestNlpOptions, error) {
	fs := flag.NewFlagSet("corpus ingest-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusIngestNlpOptions
	fs.StringVar(&opts.NlpScript, "nlp-script", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.NlpScript, "s", os.Getenv("SEGROB_NLP_SCRIPT"), "")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.BoolVar(&opts.Force, "force", false, "")
	fs.BoolVar(&opts.Force, "f", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ingest-nlp [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Process document text with NLP and store results in the corpus.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to process")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --nlp-script", "PATH", "Path to the Python NLP script (or SEGROB_NLP_SCRIPT)")
		printOpt(w, "--db", "FILE", "Path to the corpus SQLite database (or SEGROB_CORPUS_DB)")
		printOpt(w, "-f, --force", "", "Force processing even if TxtAck is false")
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
		return opts, fmt.Errorf("--db must be supplied if SEGROB_CORPUS_DB is not set")
	}

	if fs.NArg() != 1 {
		return opts, fmt.Errorf("requires exactly 1 argument (doc ID)")
	}
	opts.ID = fs.Arg(0)

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

	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus show [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Show rendered contents of a document's NLP field from the corpus staging database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-s, --start", "INDEX", "Index of the first sentence to show (default: 0)")
		printOpt(w, "-n, --number", "N", "Number of sentences to show")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, "", errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
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
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.BoolVar(&opts.Pandoc, "pandoc", false, "")
	fs.BoolVar(&opts.Pandoc, "p", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ingest-meta [options] <dir>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Scan a directory for epub files and build a corpus database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "dir", "Directory to scan for epub files")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Output SQLite file for corpus data (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusDumpTxtArgs(args []string, ui UI) (CorpusDumpTxtOptions, error) {
	fs := flag.NewFlagSet("corpus dump-txt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusDumpTxtOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.StringVar(&opts.Output, "output", "", "")
	fs.StringVar(&opts.Output, "o", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus dump-txt [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Output the txt field of a corpus document byte-exact.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusDumpNlpArgs(args []string, ui UI) (CorpusDumpNlpOptions, error) {
	fs := flag.NewFlagSet("corpus dump-nlp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusDumpNlpOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
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
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusLsArgs(args []string, ui UI) (CorpusLsOptions, error) {
	fs := flag.NewFlagSet("corpus ls", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusLsOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.BoolVar(&opts.WithNlp, "with-nlp", false, "")
	fs.BoolVar(&opts.WithNlp, "w", false, "")
	fs.BoolVar(&opts.NlpAck, "nlp-ack", false, "")
	fs.BoolVar(&opts.NlpAck, "n", false, "")
	fs.BoolVar(&opts.TxtAck, "txt-ack", false, "")
	fs.BoolVar(&opts.TxtAck, "t", false, "")
	fs.BoolVar(&opts.Ack, "ack", false, "")
	fs.BoolVar(&opts.Ack, "a", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ls [options] [filter]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List all documents in the corpus staging database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "filter", "Optional substring filter on document labels")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
		printOpt(w, "-w, --with-nlp", "", "Only list records that have NLP data")
		printOpt(w, "-n, --nlp-ack", "", "Only list records with NLP acknowledged")
		printOpt(w, "-t, --txt-ack", "", "Only list records with text acknowledged")
		printOpt(w, "-a, --ack", "", "Only list records with both NLP and text acknowledged")
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
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusPushTxtArgs(args []string, ui UI) (CorpusPushTxtOptions, error) {
	fs := flag.NewFlagSet("corpus push-txt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusPushTxtOptions
	fs.StringVar(&opts.By, "by", "", "Author of the text edit")
	fs.StringVar(&opts.Note, "note", "", "Optional note for the text edit")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "Corpus SQLite file (or SEGROB_CORPUS_DB)")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus push-txt [options] <id> <file>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Update a corpus document text from a file.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, helpArgFmt, "file", "Path to the plain text file")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--by", "NAME", "Author of the text edit")
		printOpt(w, "--note", "TEXT", "Optional note for the text edit")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fs.SetOutput(ui.Out)
			fs.Usage()
			return opts, err
		}
		return opts, err
	}

	if fs.NArg() != 2 {
		return opts, errors.New("corpus push-txt requires exactly two arguments: <id> <file>")
	}

	opts.ID = fs.Arg(0)
	opts.File = fs.Arg(1)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusAckArgs(args []string, ui UI) (CorpusAckOptions, error) {
	fs := flag.NewFlagSet("corpus ack", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusAckOptions
	fs.BoolVar(&opts.Nlp, "nlp", false, "Acknowledge NLP instead of text")
	fs.BoolVar(&opts.Nlp, "n", false, "alias for -nlp")
	fs.StringVar(&opts.By, "by", "", "Who is acknowledging")
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "Corpus SQLite file (or SEGROB_CORPUS_DB)")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ack [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Acknowledge a corpus document text or NLP.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "-n, --nlp", "", "Acknowledge NLP instead of text")
		printOpt(w, "--by", "NAME", "Who is acknowledging")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus ack requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

func parseCorpusRmArgs(args []string, ui UI) (CorpusRmOptions, error) {
	fs := flag.NewFlagSet("corpus rm", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusRmOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "Corpus SQLite file (or SEGROB_CORPUS_DB)")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus rm [options] <id>\n\n", os.Args[0])
		fmt.Fprintf(w, "  Remove a document from the corpus database.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "FILE", "Corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus rm requires exactly one argument: <id>")
	}

	opts.ID = fs.Arg(0)

	if opts.DbPath == "" {
		return opts, errors.New("corpus database must be specified via --db or SEGROB_CORPUS_DB")
	}

	return opts, nil
}

// parseCorpusSetLabelArgs parses arguments and flags for "corpus set-label".
func parseCorpusSetLabelArgs(args []string, ui UI) (CorpusSetLabelOptions, error) {
	fs := flag.NewFlagSet("corpus set-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusSetLabelOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.BoolVar(&opts.Delete, "delete", false, "")
	fs.BoolVar(&opts.Delete, "d", false, "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus set-label [options] <doc_id> <label> [<label>...]\n\n", os.Args[0])
		fmt.Fprintf(w, "  Add or remove one or more labels from a corpus document.\n")
		fmt.Fprintf(w, "  Labels are automatically normalized: lowercased, spaces and hyphens\n")
		fmt.Fprintf(w, "  replaced with underscores, commas removed.\n\n")
		fmt.Fprintf(w, "  Recommended format: prefix:value (e.g., creator:garcía_lorca).\n")
		fmt.Fprintf(w, "  The following prefixes are displayed as columns by 'corpus ls':\n")
		fmt.Fprintf(w, "    creator:  title:  translator:  date:  language:\n\n")
		fmt.Fprintf(w, "Arguments:\n")
		fmt.Fprintf(w, helpArgFmt, "doc_id", "ID of the document")
		fmt.Fprintf(w, helpArgFmt, "label", "One or more labels to add/remove")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "PATH", "Path to corpus SQLite file (or SEGROB_CORPUS_DB)")
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
		return opts, errors.New("corpus set-label requires at least two arguments: <doc_id> and one or more <label>")
	}

	opts.DocID = fs.Arg(0)
	opts.Labels = fs.Args()[1:]

	if opts.DbPath == "" {
		return opts, errors.New("no document source specified (use --db or SEGROB_CORPUS_DB)")
	}

	return opts, nil
}

// parseCorpusLsLabelArgs parses arguments and flags for "corpus ls-label".
func parseCorpusLsLabelArgs(args []string, ui UI) (CorpusLsLabelOptions, error) {
	fs := flag.NewFlagSet("corpus ls-label", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts CorpusLsLabelOptions
	fs.StringVar(&opts.DbPath, "db", os.Getenv("SEGROB_CORPUS_DB"), "")
	fs.StringVar(&opts.Match, "match", "", "")
	fs.StringVar(&opts.Match, "m", "", "")

	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintf(w, "Usage: %s corpus ls-label [options] [id]\n\n", os.Args[0])
		fmt.Fprintf(w, "  List labels in the corpus.\n")
		fmt.Fprintf(w, "  When no id is given, lists all unique labels across the corpus.\n")
		fmt.Fprintf(w, "\nArguments:\n")
		fmt.Fprintf(w, helpArgFmt, "id", "Document ID to list labels for (omit to list all)")
		fmt.Fprintf(w, "\nOptions:\n")
		printOpt(w, "--db", "PATH", "Path to corpus SQLite file (or SEGROB_CORPUS_DB)")
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

	// Capture optional positional argument
	if fs.NArg() > 0 {
		if fs.NArg() > 1 {
			return opts, errors.New("corpus ls-label accepts at most one argument: [id]")
		}
		opts.ID = fs.Arg(0)
	}

	if opts.DbPath == "" {
		return opts, errors.New("no document source specified (use --db or SEGROB_CORPUS_DB)")
	}

	return opts, nil
}
