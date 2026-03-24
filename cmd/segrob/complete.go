package main

import (
	"fmt"
	"strings"
)

var commands = []string{
	"corpus",
	"live",
	"bash",
	"version",
	"help",
}

var corpusSubcommands = []string{
	"init",
	"ls",
	"show",
	"ack",
	"rm",
	"backup",
	"publish",
	"publish-label",
	"dump-txt",
	"dump-nlp",
	"ingest-nlp",
	"ingest-meta",
	"push-txt",
	"ls-label",
	"set-label",
	"ls-topic",
	"show-topic",
	"edit",
	"import-topic",
	"export-topic",
}

var liveSubcommands = []string{
	"ls",
	"find",
	"find-topics",
	"unpublish",
	"init",
	"ls-topic",
	"show",
	"show-sent",
	"show-topic",
	"query",
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

	// Second-level completion for group commands
	if cursorIndex == commandIndex+1 {
		command := args[commandIndex]
		lastWord := args[cursorIndex]
		var subcommands []string
		switch command {
		case "corpus":
			subcommands = corpusSubcommands
		case "live":
			subcommands = liveSubcommands
		}
		var completions []string
		for _, s := range subcommands {
			if strings.HasPrefix(s, lastWord) {
				completions = append(completions, s)
			}
		}
		return completions
	}

	return nil
}
