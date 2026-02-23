package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type docMetaDTO struct {
	Source string   `toml:"source"`
	Labels []string `toml:"labels"`
}

func importMetaCommand(p *Pool, opts ImportMetaOptions, ui UI) error {
	data, err := os.ReadFile(opts.Filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", opts.Filename, err)
	}

	var doc docMetaDTO
	if err := toml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("failed to decode TOML: %w", err)
	}

	// Derived source if missing
	if doc.Source == "" {
		base := filepath.Base(opts.Filename)
		doc.Source = strings.TrimSuffix(base, ".meta.toml")
	}

	repo, err := NewDocRepository(p, opts.DbPath)
	if err != nil {
		return err
	}

	if err := repo.WriteMeta(doc.Source, doc.Labels); err != nil {
		return fmt.Errorf("failed to write meta: %w", err)
	}

	return nil
}
