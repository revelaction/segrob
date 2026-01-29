package main

import (
	"fmt"
	"path/filepath"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func importDocCommand(opts ImportDocOptions, ui UI) error {
	src, err := filesystem.NewDocHandler(opts.From)
	if err != nil {
		return err
	}
	// We only need the list of filenames
	if err := src.LoadList(); err != nil {
		return err
	}

	pool, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := zombiezen.CreateDocTables(pool); err != nil {
		return fmt.Errorf("failed to create docs table: %w", err)
	}

	dst := zombiezen.NewDocHandler(pool)

	docs, err := src.List()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(ui.Err, "Importing %d docs from %s to %s...\n", len(docs), opts.From, opts.To)

	count := 0
	for _, docMeta := range docs {
		// Read document directly from disk to avoid keeping everything in memory
		docPath := filepath.Join(opts.From, docMeta.Title)
		doc, err := filesystem.ReadDoc(docPath)
		if err != nil {
			return fmt.Errorf("failed to read doc %s: %w", docMeta.Title, err)
		}

		// Ensure Title is set (fallback to filename if missing in JSON)
		if doc.Title == "" {
			doc.Title = docMeta.Title
		}

		if err := dst.WriteDoc(doc); err != nil {
			return fmt.Errorf("failed to write doc %s: %w", docMeta.Title, err)
		}
		count++
		_, _ = fmt.Fprintf(ui.Err, "[%d/%d] Imported %s\n", count, len(docs), docMeta.Title)
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully imported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}