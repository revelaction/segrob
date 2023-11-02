package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/file"
	"github.com/revelaction/segrob/match"
	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/stat"
	"github.com/revelaction/segrob/topic"

	"github.com/gosuri/uiprogress"
	"github.com/urfave/cli/v2"
)

// Some commands need this for autocomplation, validation. Lazy loaded
var TopicNames []string
var DocNames []string

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "[segrob] %v\n", err)
	os.Exit(1)
}

// Topic strutcs

func main() {

	app := &cli.App{}
	app.Name = "segrob"
	app.EnableBashCompletion = true
	app.UseShortOptionHandling = true
	//app.Version = build.Version() + " commit=" + build.Commit
	app.Usage = "sentence dictionary based on nlp topics"
	app.Flags = []cli.Flag{
		&cli.StringSliceFlag{
			Name:    "label",
			Aliases: []string{"l"},
			Usage:   "Only scan those token files that match the labels (constains)",
		},
		&cli.BoolFlag{
			Name:    "no-color",
			Aliases: []string{"c"},
			Usage:   "Show matched sentences without formating (color)",
		},
		&cli.BoolFlag{
			Name:    "no-prefix",
			Aliases: []string{"x"},
			Usage:   "Show matched sentences without prefixes with metadata",
		},
		&cli.IntFlag{
			Name:    "doc",
			Aliases: []string{"d"},
			Usage:   "Limit searched to the doc specied by this number",
		},
		&cli.IntFlag{
			Name:    "sent",
			Aliases: []string{"s"},
			Usage:   "Limit searched to the sentence specied by this number. Needs --doc",
		},
		&cli.IntFlag{
			Name:    "nmatches",
			Aliases: []string{"n"},
			Usage:   "Only show matched sentences with score greater than this number",
		},
		&cli.GenericFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Value: &EnumValue{
				Enum:    render.SupportedFormats(),
				Default: render.Defaultformat,
			},
			Usage: "Show whole sentence (all), only sorrounding of matched words (part) or only matches words (lemma)",
		},
		&cli.StringFlag{
			Name:    "expr",
			Aliases: []string{"e"},
			Usage:   "train: Select words in prompt that match this expresion token (only Tag supported)",
		},
		&cli.BoolFlag{
			Name:    "token",
			Aliases: []string{"t"},
			Usage:   "doc: Show Doc (parts) as json Lesson document",
		},
	}

	// doc command
	command := &cli.Command{
		Name:     "doc",
		Action:   docAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// sentence command
	command = &cli.Command{
		Name:     "sentence",
		Action:   sentenceAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// topic command
	command = &cli.Command{
		Name:     "topics",
		Action:   topicsAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// expr command
	command = &cli.Command{
		Name:     "expr",
		Action:   exprAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// query command
	command = &cli.Command{
		Name:     "query",
		Action:   queryAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// refine command
	command = &cli.Command{
		Name:     "edit",
		Action:   editAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	// Get all topics
	th := file.NewTopicHandler()

	topicNames, err := th.Names()
	if err != nil {
		fatal(err)
	}

	// topic command
	command = &cli.Command{
		Name:     "topic",
		Action:   topicAction,
		Category: "general",
		BashComplete: func(c *cli.Context) {
			// This will complete if no args are passed
			if c.NArg() > 0 {
				return
			}

			for _, tn := range topicNames {
				fmt.Println(tn)
			}
		},
	}

	app.Commands = append(app.Commands, command)

	// stat command
	command = &cli.Command{
		Name:     "stat",
		Action:   statAction,
		Category: "general",
	}

	app.Commands = append(app.Commands, command)

	if err := app.Run(os.Args); err != nil {
		fatal(err)
	}
}

// Query command
func queryAction(ctx *cli.Context) error {

	// Load docs
	fhr, err := file.NewDocHandler()
	docLib, err := docLibrary(fhr)
	if err != nil {
		return err
	}

	th := file.NewTopicHandler()
	topicLib, err := topicLibrary(th)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.HasColor = !ctx.Bool("no-color")
	r.HasPrefix = !ctx.Bool("no-prefix")
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = ctx.String("format")
	r.NumMatches = ctx.Int("nmatches")

	// now present the REPL and prepare for topic in the REPL
	t := query.NewHandler(docLib, topicLib, r)
	return t.Run()
}

func docLibrary(fhr *file.DocHandler) (sent.Library, error) {
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
		//r.AddDocName(docId, doc.Title)
		//matcher.Match(doc)
		library = append(library, doc)

		bar.Incr()
	}

	// stop rendering
	uiprogress.Stop()

	return library, nil
}

// topicLibrary retrieves all expressions of all topic files
func topicLibrary(th *file.TopicHandler) (topic.Library, error) {

	topicNames, err := th.Names()
	if err != nil {
		return nil, err
	}

	var library topic.Library

	for _, name := range topicNames {

		tp, err := th.Topic(name)
		if err != nil {
			fmt.Fprintf(os.Stdout, "✍  %s %s \n", err, name)
			return nil, err
		}

		library = append(library, tp)
	}

	return library, nil
}

func matchDocs(matcher *match.Matcher, ctx *cli.Context) error {

	if ctx.IsSet("sent") {
		if !ctx.IsSet("doc") {
			return errors.New("--sent flag given but no --doc")
		}
	}

	r := render.NewRenderer()
	r.HasColor = !ctx.Bool("no-color")
	r.HasPrefix = !ctx.Bool("no-prefix")
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = ctx.String("format")
	r.NumMatches = ctx.Int("nmatches")

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	switch {
	case ctx.IsSet("doc"):
		docId := ctx.Int("doc")
		doc, err := fhr.Doc(docId)
		if err != nil {
			return err
		}

		doc.Id = docId

		if ctx.IsSet("sent") {
			doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[ctx.Int("sent")]}}
		}

		matcher.Match(doc)

	default:
		docNames, err := fhr.Names()
		if err != nil {
			return err
		}

		for docId, name := range docNames {

			doc, err := fhr.DocForName(name)
			if err != nil {
				return err
			}

			if !hasLabels(doc.Labels, ctx.StringSlice("label")) {
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

func statAction(ctx *cli.Context) error {

	ln := ctx.Args().Len()

	// No arguments
	if ln == 0 {
		return errors.New("Usage [<docId> <sentenceId>, <label>]")
	}

	docId, err := strconv.ParseInt(ctx.Args().First(), 10, 64)
	if err != nil {
		return err
	}

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	doc, err := fhr.Doc(int(docId))
	if err != nil {
		return err
	}

	if ln == 2 {
		numSentence, err := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}
		// rewrite
		doc = sent.Doc{Tokens: [][]sent.Token{doc.Tokens[int(numSentence)]}}
	}

	hdl := stat.NewHandler()
	hdl.Aggregate(doc)

	stats := hdl.Get()
	fmt.Fprintf(os.Stdout, "Num sentences %d, num tokens per sentence %d\n", stats.NumSentences, stats.TokensPerSentenceMean)

	return nil
}

func exprAction(ctx *cli.Context) error {

	argsErr := errors.New("Usage <topic expr item> ...")
	if ctx.Args().Len() < 1 {
		return argsErr
	}

	// parse the expr expresion
	expr, parseErr := topic.Parse(ctx.Args().Slice())
	if parseErr != nil {
		return parseErr
	}

	matcher := match.NewMatcher(topic.Topic{})
	matcher.AddTopicExpr(expr)
	err := matchDocs(matcher, ctx)
	if err != nil {
		return err
	}

	return nil
}

func docAction(ctx *cli.Context) error {

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	ln := ctx.Args().Len()

	if ln == 0 {
		docNames, err := fhr.Names()
		if err != nil {
			return err
		}

		for docId, name := range docNames {
			fmt.Fprintf(os.Stdout, "📖 %d %s \n", docId, name)
		}

		return nil
	}

	// take the first and consider docId or doc name `match`, read file and
	// iterate for sentence
	first := ctx.Args().First()
	docId, err := strconv.ParseInt(first, 10, 64)
	var doc sent.Doc
	if err != nil {
		// could not parse as integer, try to match word
		doc, err = fhr.DocForMatch(first)
		if err != nil {
			return err
		}
	} else {
		doc, err = fhr.Doc(int(docId))
		if err != nil {
			return err
		}
	}

	var startSentence, endSentence int64
	switch {
	case ln == 2:
		startSentence, err = strconv.ParseInt(ctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

	case ln > 2:
		startSentence, err = strconv.ParseInt(ctx.Args().Get(1), 10, 64)
		if err != nil {
			return err
		}

		endSentence, err = strconv.ParseInt(ctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		if ctx.Bool("token") {
			return nil
		}
	}

	for i, sentence := range doc.Tokens {
		if i < int(startSentence) {
			continue
		}

		if endSentence > 0 && i > int(endSentence) {
			continue
		}

		r := render.NewRenderer()
		r.HasColor = false
		prefix := fmt.Sprintf("✍  %d-%d ", docId, i)
		r.Sentence(sentence, prefix)
	}

	return nil
}

func sentenceAction(ctx *cli.Context) error {

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	if ctx.Args().Len() < 2 {
		return errors.New("Usage <docId> <sentenceId> [offset]")
	}

	docId, err := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
	if err != nil {
		return err
	}

	sentId, err := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
	if err != nil {
		return err
	}

	doc, err := fhr.Doc(int(docId))
	if err != nil {
		return err
	}

	s := doc.Tokens[sentId]
	r := render.NewRenderer()
	r.HasColor = false
	prefix := fmt.Sprintf("✍  %d-%d ", docId, sentId)
	r.Sentence(s, prefix)
	fmt.Fprintln(os.Stdout)

	var offset int64 = 0
	if ctx.Args().Len() > 2 {
		offset, err = strconv.ParseInt(ctx.Args().Get(2), 10, 64)
		if err != nil {
			return err
		}

		// check len
		if int(offset) > len(s) {
			return errors.New("offset is greater than lengh of sentence. Usage <docId> <sentenceId> [offset]")
		}
	}

	for _, token := range s[int(offset):] {
		// print
		fmt.Fprintf(os.Stdout, "%20q %15q %8s %6d %6d %8s %s\n", token.Text, token.Lemma, token.Pos, token.Id, token.Head, token.Dep, token.Tag)
	}

	return nil
}

func editAction(ctx *cli.Context) error {

	th := file.NewTopicHandler()

	topicLib, err := topicLibrary(th)
	if err != nil {
		return err
	}

	hdl := edit.NewHandler(topicLib, th)
	return hdl.Run()
}

func topicsAction(ctx *cli.Context) error {
	if ctx.Args().Len() != 2 {
		return errors.New("Usage <docId> <sentenceId>")
	}

	docId, err := strconv.ParseInt(ctx.Args().Get(0), 10, 64)
	if err != nil {
		return err
	}

	sentId, err := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
	if err != nil {
		return err
	}

	fhr, err := file.NewDocHandler()
	if err != nil {
		return err
	}

	doc, err := fhr.Doc(int(docId))
	if err != nil {
		return err
	}

	s := doc.Tokens[sentId]
	doc = sent.Doc{Tokens: [][]sent.Token{s}}

	r := render.NewRenderer()
	r.HasColor = false

	prefix := fmt.Sprintf("%54s", render.Yellow256+render.Off) + "✍  "
	r.Sentence(s, prefix)
	fmt.Println()

	th := file.NewTopicHandler()

	allTopics, err := th.All()
	if err != nil {
		return err
	}

	r.HasColor = true
	r.HasPrefix = true
	r.PrefixDocFunc = render.PrefixFuncEmpty
	r.Format = ctx.String("format")

	for _, tp := range allTopics {

		matcher := match.NewMatcher(tp)
		matcher.Match(doc)
		res := matcher.Sentences()

		if len(res) == 0 {
			continue
		}

		r.Match(res)
	}

	return nil
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

// topicAction prints the expressions of a topic
func topicAction(ctx *cli.Context) error {

	fhr := file.NewTopicHandler()

	ln := ctx.Args().Len()

	// No arguments
	if ln == 0 {
		topicNames, err := fhr.Names()
		if err != nil {
			return err
		}

		for topicId, name := range topicNames {
			fmt.Fprintf(os.Stdout, "📖 %d %s \n", topicId, name)
		}

		return nil
	}

	first := ctx.Args().First()
	tp, err := fhr.Topic(first)
	if err != nil {
		return err
	}

	r := render.NewRenderer()
	r.Topic(tp.Exprs)
	return nil
}

// urfave enum flags https://github.com/urfave/cli/issues/602
type EnumValue struct {
	Enum     []string
	Default  string
	selected string
}

func (e *EnumValue) Set(value string) error {
	for _, enum := range e.Enum {
		if enum == value {
			e.selected = value
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", strings.Join(e.Enum, ", "))
}

func (e EnumValue) String() string {
	if e.selected == "" {
		return e.Default
	}
	return e.selected
}

func topicNames() []string {

	if len(TopicNames) > 0 {
		return TopicNames
	}

	th := file.NewTopicHandler()

	names, err := th.Names()
	if err != nil {
		fatal(err)
	}
	return names
}

func docNames() []string {

	if len(DocNames) > 0 {
		return DocNames
	}

	fhr, err := file.NewDocHandler()
	if err != nil {
		fatal(err)
	}

	names, err := fhr.Names()
	if err != nil {
		fatal(err)
	}

	return names
}
