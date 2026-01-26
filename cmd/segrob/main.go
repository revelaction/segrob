package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/file"
	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
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
		opts, err := parseQueryArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return queryCommand(opts, ui)

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
		name, err := parseTopicArgs(args, ui)
		if err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil
			}
			return err
		}
		return topicCommand(name, ui)

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
	}

	return fmt.Errorf("unknown command: %s", cmd)
}

// Query command
func queryCommand(opts QueryOptions, ui UI) error {

	// Load docs
	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}
	docLib, err := docLibrary(fhr, ui)
	if err != nil {
		return err
	}

	th := file.NewTopicHandler(file.TopicDir)
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
	t := query.NewHandler(docLib, topicLib, r)
	return t.Run()
}

func docLibrary(fhr *file.DocHandler, ui UI) (sent.Library, error) {
	docNames, err := fhr.Names()
	if err != nil {
		return nil, err
	}

	var library sent.Library

	// Start progress indicator
	uiprogress.Start()                      // start rendering
	bar := uiprogress.AddBar(len(docNames)) // Add a new bar
	bar.AppendCompleted()
	bar.PrependElapsed()
	bar.Set(1)
	// Append Doc name to the progress bar
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		return docNames[b.Current()-1]
	})

	for docId, name := range docNames {

		doc, err := fhr.DocForName(name)
		if err != nil {
			return nil, err
		}

		// Add Here the Id.
		doc.Id = docId
		library = append(library, doc)

		bar.Incr()
	}

	// stop rendering
	uiprogress.Stop()

	return library, nil
}

// topicLibrary retrieves all expressions of all topic files
func topicLibrary(th topic.TopicReader, ui UI) (topic.Library, error) {

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

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	if opts.Doc != nil {
		docId := *opts.Doc
		doc, err := fhr.Doc(docId)
		if err != nil {
			return err
		}

		doc.Id = docId

		if opts.Sent != nil {
			doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[*opts.Sent]}}
		}

		matcher.Match(doc)

	} else {
		docNames, err := fhr.Names()
		if err != nil {
			return err
		}

		for docId, name := range docNames {

			doc, err := fhr.DocForName(name)
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
		return errors.New("SQLite backend not yet implemented")
	}

	th := file.NewTopicHandler(opts.TopicPath)

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
func topicCommand(name string, ui UI) error {

	fhr := file.NewTopicHandler(file.TopicDir)

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
