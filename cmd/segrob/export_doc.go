package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gosuri/uiprogress"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func exportDocCommand(opts ExportDocOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.From)
	if err != nil {
		return err
	}
	defer pool.Close()
	src := zombiezen.NewDocHandler(pool)

	// Ensure target directory exists
	if err := os.MkdirAll(opts.To, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	docs, err := src.List()
	if err != nil {
		return err
	}

	uiprogress.Start()
	bar := uiprogress.AddBar(len(docs))
	bar.AppendCompleted()
	bar.PrependElapsed()

	count := 0
	for _, docMeta := range docs {
		doc, err := src.DocForName(docMeta.Title)
		if err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to read doc %s: %w", docMeta.Title, err)
		}

		// Write to JSON
		data, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			uiprogress.Stop()
			return err
		}

		targetPath := filepath.Join(opts.To, docMeta.Title)
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
		count++
		bar.Incr()
	}
	uiprogress.Stop()

	fmt.Fprintf(ui.Out, "Successfully exported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}
