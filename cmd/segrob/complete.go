package main

import (
	"fmt"
	"strings"
)

var commands = []string{
	"doc",
	"sentence",
	"topics",
	"expr",
	"query",
	"edit",
	"topic",
	"stat",
	"bash",
	"help",
}

// completeCommand handles the autocompletion requests triggered by the bash completion script.
func completeCommand(args []string, ui UI) error {
	completions := getCompletions(args)
	for _, c := range completions {
		_, _ = fmt.Fprintln(ui.Out, c)
	}
	return nil
}

func getCompletions(args []string) []string {
	if len(args) < 1 {
		return nil
	}

	// 1. Find the index where the subcommand should be
	// args[0] is "segrob" (binary name from COMP_WORDS[0])
	commandIndex := 1
	cursorIndex := len(args) - 1

	// 2. Decide what to complete based on cursor position relative to command position
	if cursorIndex == commandIndex {
		// User is typing the command itself
		lastWord := args[cursorIndex]
		var completions []string
		for _, c := range commands {
			if strings.HasPrefix(c, lastWord) {
				completions = append(completions, c)
			}
		}
		return completions
	}

	return nil
}
