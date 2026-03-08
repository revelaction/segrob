package main

import (
	"os"

	"github.com/revelaction/segrob/query"
	"github.com/revelaction/segrob/render"
	"github.com/revelaction/segrob/storage"
	"golang.org/x/term"
)

// Query command
func liveQueryCommand(dr storage.DocRepository, tr storage.TopicRepository, opts LiveQueryOptions, ui UI) error {

	// Terminal Reset
	//
	// The issue occurs because go-prompt puts your terminal into Raw Mode (to
	// handle custom keybinds and colors) but fails to restore it to Cooked
	// Mode (canonical mode) upon exit. When the terminal is left in Raw Mode,
	// it often disables local echo (typing is invisible) and carriage
	// returns.
	// For interactive commands, we save the terminal state (Cooked Mode)
	// and strictly restore it when the function returns.
	fd := int(os.Stdin.Fd())
	if state, err := term.GetState(fd); err == nil {
		defer term.Restore(fd, state)
	}

	topicLib, err := tr.ReadAll()
	if err != nil {
		return err
	}

	r := render.NewCLIRenderer()
	r.HasColor = !opts.NoColor
	r.HasPrefix = !opts.NoPrefix
	r.PrefixTopicFunc = render.PrefixFuncEmpty
	r.Format = opts.Format
	r.NumMatches = opts.NMatches

	// now present the REPL and prepare for topic in the REPL
	t := query.NewHandler(dr, topicLib, r, opts.Labels)
	return t.Run()
}
