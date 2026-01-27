package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/file"
	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"github.com/revelaction/segrob/topic"

	"github.com/gosuri/uiprogress"
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
		opts, source, sentId, isFile, err := parseTopicsArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return topicsCommand(opts, source, sentId, isFile, ui)

	case "expr":
		opts, cmdArgs, err := parseExprArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return exprCommand(opts, cmdArgs, ui)

	case "query":
		opts, isFile, err := parseQueryArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return queryCommand(opts, isFile, ui)

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

// Query command
func queryCommand(opts QueryOptions, isFile bool, ui UI) error {

	if isFile {
		// This flag might be obsolete now with auto-detection
	}

	// Load docs
	dr, err := getDocHandler(opts.DocPath)
	if err != nil {
		return err
	}

	// Restore progress bar behavior for filesystem (legacy parity)
	if fsHandler, ok := dr.(*filesystem.DocHandler); ok {
		uiprogress.Start()
		bar := uiprogress.AddBar(1) // Placeholder, updated in callback
		bar.AppendCompleted()
		bar.PrependElapsed()

		var currentName string
		bar.AppendFunc(func(b *uiprogress.Bar) string {
			return currentName
		})

		err = fsHandler.LoadWithCallback(func(total int, name string) {
			if bar.Total <= 1 {
				bar.Total = total
				bar.Set(0)
			}
			currentName = name
			bar.Incr()
		})
		uiprogress.Stop()

		if err != nil {
			return err
		}
	}

	th, err := getTopicHandler(opts.TopicPath)
	if err != nil {
		return err
	}
	topicLib, err := topicLibrary(th, ui)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	// now present the REPL and prepare for topic in the REPL
	t := query.NewHandler(dr, topicLib, r)
	return t.Run()
}

// topicLibrary retrieves all expressions of all topic files
func topicLibrary(th storage.TopicReader, ui UI) (topic.Library, error) {

	topicNames, err := th.Names()
	if err != nil {
		return nil, err
	}

	var library topic.Library

	for _, name := range topicNames {

		tp, err := th.Topic(name)
		if err != nil {
			fmt.Fprintf(ui.Out, "✍  %s %s \n", err, name)
			return nil, err
		}

		library = append(library, tp)
	}

	return library, nil
}

func matchDocs(matcher *match.Matcher, opts ExprOptions, ui UI) error {

	if opts.Sent != nil {
		if opts.Doc == nil {
			return errors.New("--sent flag given but no --doc")
		}
	}

	r := render.NewRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	dr, err := getDocHandler("")
	if err != nil {
		return err
	}

	if fsHandler, ok := dr.(*filesystem.DocHandler); ok {
		if err := fsHandler.LoadWithCallback(nil); err != nil {
			return err
		}
	}

	if opts.Doc != nil {
		docId := *opts.Doc
		doc, err := dr.Doc(docId)
		if err != nil {
			return err
		}

		doc.Id = docId

		if opts.Sent != nil {
			doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*opts.Sent]}}
		}

		matcher.Match(doc)

	} else {
		docNames, err := dr.Names()
		if err != nil {
			return err
		}

		for docId, name := range docNames {

			doc, err := dr.DocForName(name)
			if err != nil {
				return err
			}

			if !hasLabels(doc.Labels, opts.Labels) {
				continue
			}

			doc.Id = docId
			r.AddDocName(docId, doc.Title)
			matcher.Match(doc)
		}
	}

	result := matcher.Sentences()

	r.Match(result)
	return nil
}

func exprCommand(opts ExprOptions, args []string, ui UI) error {

	// args is guaranteed to have at least 1 element by parseExprArgs
	// parse the expr expression
	expr, parseErr := topic.Parse(args)
	if parseErr != nil {
		return parseErr
	}

	matcher := match.NewMatcher(topic.Topic{})
	matcher.AddTopicExpr(expr)
	err := matchDocs(matcher, opts, ui)
	if err != nil {
		return err
	}

	return nil
}

func editCommand(opts EditOptions, isFile bool, ui UI) error {

	if isFile {
		// This flag might be obsolete now with auto-detection
	}

	th, err := getTopicHandler(opts.TopicPath)
	if err != nil {
		return err
	}

	topicLib, err := topicLibrary(th, ui)
	if err != nil {
		return err
	}

	hdl := edit.NewHandler(topicLib, th, th)
	return hdl.Run()
}

func hasLabels(fileLabels, cmdLabels []string) bool {
	// No command line labels to match
	if nil == cmdLabels {
		return true
	}

	for _, label := range cmdLabels {

		isLabel := false
		for _, l := range fileLabels {
			if strings.Contains(l, label) {
				isLabel = true
			}
		}

		if !isLabel {
			return false
		}
	}

	return true
}

// topicCommand prints the expressions of a topic
func topicCommand(opts TopicOptions, name string, isFile bool, ui UI) error {

	if isFile {
		// This flag might be obsolete now with auto-detection
	}

	fhr, err := getTopicHandler(opts.TopicPath)
	if err != nil {
		return err
	}

	// No name provided (list all)
	if name == "" {
		topicNames, err := fhr.Names()
		if err != nil {
			return err
		}

		for topicId, name := range topicNames {
			fmt.Fprintf(ui.Out, "📖 %d %s \n", topicId, name)
		}

		return nil
	}

	tp, err := fhr.Topic(name)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.Topic(tp.Exprs)
	return nil
}

func importTopicCommand(opts ImportTopicOptions, ui UI) error {
	src := filesystem.NewTopicHandler(opts.From)
	pool, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := zombiezen.Setup(pool); err != nil {
		return fmt.Errorf("failed to setup sqlite database: %w", err)
	}

	dst := zombiezen.NewTopicHandler(pool)

	topics, err := src.All()
	if err != nil {
		return err
	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to import topic %s: %w", tp.Name, err)
		}
	}

	fmt.Fprintf(ui.Out, "Successfully imported %d topics from %s to %s\n", len(topics), opts.From, opts.To)
	return nil
}

func exportTopicCommand(opts ExportTopicOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return err
	}
	defer pool.Close()
	src := zombiezen.NewTopicHandler(pool)

	dst := filesystem.NewTopicHandler(opts.To)

	topics, err := src.All()
	if err != nil {
		return err
	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to export topic %s: %w", tp.Name, err)
		}
	}

	fmt.Fprintf(ui.Out, "Successfully exported %d topics from %s to %s\n", len(topics), opts.From, opts.To)
	return nil
}

func getTopicHandler(path string) (storage.TopicRepository, error) {
	info, err := os.Stat(path)
	if err != nil {
		// Path doesn't exist, assume new SQLite DB if it looks like a file path
		pool, err := zombiezen.NewPool(path)
		if err != nil {
			return nil, err
		}
		return zombiezen.NewTopicHandler(pool), nil
	}

	if info.IsDir() {
		return filesystem.NewTopicHandler(path), nil
	}

	// Non-directory = SQLite file
	pool, err := zombiezen.NewPool(path)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewTopicHandler(pool), nil
}

func getDocHandler(path string) (storage.DocRepository, error) {
	if path == "" {
		path = file.TokenDir
	}

	info, err := os.Stat(path)
	if err != nil {
		// Path doesn't exist, assume new SQLite DB
		pool, err := zombiezen.NewPool(path)
		if err != nil {
			return nil, err
		}
		return zombiezen.NewDocHandler(pool), nil
	}

	if info.IsDir() {
		return filesystem.NewDocHandler(path), nil
	}

	// Non-directory = SQLite file
	pool, err := zombiezen.NewPool(path)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewDocHandler(pool), nil
}

func importDocCommand(opts ImportDocOptions, ui UI) error {
	src := filesystem.NewDocHandler(opts.From)
	pool, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := zombiezen.SetupDocs(pool); err != nil {
		return fmt.Errorf("failed to setup sqlite docs database: %w", err)
	}

	dst := zombiezen.NewDocHandler(pool)

	fmt.Fprintf(ui.Out, "Reading docs from %s...\n", opts.From)
	names, err := src.Names()
	if err != nil {
		return err
	}

	uiprogress.Start()
	bar := uiprogress.AddBar(len(names))
	bar.AppendCompleted()
	bar.PrependElapsed()

	count := 0
	for _, name := range names {
		doc, err := src.DocForName(name)
		if err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to read doc %s: %w", name, err)
		}

		if err := dst.WriteDoc(doc); err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to write doc %s: %w", name, err)
		}
		count++
		bar.Incr()
	}
	uiprogress.Stop()

	fmt.Fprintf(ui.Out, "Successfully imported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}

func exportDocCommand(opts ExportDocOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return err
	}
	defer pool.Close()
	src := zombiezen.NewDocHandler(pool)

	// Ensure target directory exists
	if err := os.MkdirAll(opts.To, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	names, err := src.Names()
	if err != nil {
		return err
	}

	uiprogress.Start()
	bar := uiprogress.AddBar(len(names))
	bar.AppendCompleted()
	bar.PrependElapsed()

	count := 0
	for _, name := range names {
		doc, err := src.DocForName(name)
		if err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to read doc %s: %w", name, err)
		}

		// Write to JSON
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			uiprogress.Stop()
			return err
		}

		targetPath := filepath.Join(opts.To, name)
		if err := ioutil.WriteFile(targetPath, data, 0644); err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
		count++
		bar.Incr()
	}
	uiprogress.Stop()

	fmt.Fprintf(ui.Out, "Successfully exported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}
