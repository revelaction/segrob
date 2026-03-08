package main

import (
	"os"

	"github.com/revelaction/segrob/edit"
	"github.com/revelaction/segrob/storage"
	"golang.org/x/term"
)

func editCommand(tr storage.TopicRepository, opts EditOptions, ui UI) error {

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

	hdl := edit.NewHandler(topicLib, tr, tr)
	return hdl.Run()
}
