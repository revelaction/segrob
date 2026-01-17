package main

import (
	"fmt"
)

const complete = `#! /bin/bash

_segrob_autocomplete() {
    local cur
    
    # Try to initialize using bash-completion if available
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -n "=:" 2>/dev/null
    fi
    
    # Fallback if cur is not set (e.g. _init_completion failed or missing)
    if [[ -z "$cur" ]]; then
        cur="${COMP_WORDS[COMP_CWORD]}"
    fi

    # call segrob complete with all words
    # we use -- to separate flags for "complete" command if we added any, 
    # but more importantly to pass the user's command line safely.
    local suggestions=$(segrob complete -- "${COMP_WORDS[@]}")
    
    if [ $? -eq 0 ]; then
        COMPREPLY=( $(compgen -W "$suggestions" -- "$cur") )
    fi
}

complete -F _segrob_autocomplete segrob
`

func bashCommand(ui UI) error {
	_, err := fmt.Fprint(ui.Out, complete)
	return err
}
