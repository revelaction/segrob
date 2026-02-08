package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	sent "github.com/revelaction/segrob/sentence"
)

type migrateOptions struct {
	From string
	To   string
}

// legacyDoc mirrors the old JSON structure: tokens: [][]Token
type legacyDoc struct {
	Labels []string          `json:"labels"`
	Tokens [][]sent.Token     `json:"tokens"`
}

func migrateCommand(opts migrateOptions, ui UI) error {
	files, err := os.ReadDir(opts.From)
	if err != nil {
		return fmt.Errorf("failed to read source directory: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(opts.To, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		path := filepath.Join(opts.From, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", file.Name(), err)
		}

		var old legacyDoc
		if err := json.Unmarshal(data, &old); err != nil {
			return fmt.Errorf("failed to unmarshal legacy doc %s: %w", file.Name(), err)
		}

		newDoc := sent.Doc{
			Title:  file.Name(),
			Labels: old.Labels,
		}

		for i, tokens := range old.Tokens {
			newDoc.Sentences = append(newDoc.Sentences, sent.Sentence{
				Id:     i,
				Tokens: tokens,
			})
		}

		// Write to new JSON format
		newData, err := json.MarshalIndent(newDoc, "", "  ")
		if err != nil {
			return err
		}

		targetPath := filepath.Join(opts.To, file.Name())
		if err := os.WriteFile(targetPath, newData, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		fmt.Fprintf(ui.Err, "Migrated %s to %s\n", file.Name(), targetPath)
	}

	return nil
}