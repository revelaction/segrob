package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func exportDocCommand(opts ExportDocOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return err
	}
	defer pool.Close()
	src := zombiezen.NewDocStore(pool)

	// Ensure target directory exists
	if err := os.MkdirAll(opts.To, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	docs, err := src.List("")
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(ui.Err, "Exporting %d docs from %s to %s...\n", len(docs), opts.From, opts.To)

	count := 0
	for _, docMeta := range docs {
		doc, err := src.Read(docMeta.Id)
		if err != nil {
			return fmt.Errorf("failed to read doc %s (id %d): %w", docMeta.Title, docMeta.Id, err)
		}

		// Ensure title is set in the exported document
		doc.Title = docMeta.Title

		// Write to JSON
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}

		targetPath := filepath.Join(opts.To, docMeta.Title)
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
		count++
		_, _ = fmt.Fprintf(ui.Err, "[%d/%d] Exported %s\n", count, len(docs), docMeta.Title)
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully exported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}
