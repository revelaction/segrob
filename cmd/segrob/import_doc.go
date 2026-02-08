package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func importDocCommand(opts ImportDocOptions, ui UI) error {
	src, err := filesystem.NewDocStore(opts.From)
	if err != nil {
		return err
	}

	pool, err := zombiezen.NewPool(opts.To)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := zombiezen.CreateSchemas(pool, "docs.sql"); err != nil {
		return fmt.Errorf("failed to setup doc tables: %w", err)
	}

	dst := zombiezen.NewDocStore(pool)

	docs, err := src.List()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(ui.Err, "Importing %d docs from %s to %s...\n", len(docs), opts.From, opts.To)

	count := 0
	for _, docMeta := range docs {
		doc, err := src.Read(docMeta.Id)
		if err != nil {
			return fmt.Errorf("failed to read doc %s: %w", docMeta.Title, err)
		}

		if err := dst.Write(doc); err != nil {
			return fmt.Errorf("failed to write doc %s: %w", docMeta.Title, err)
		}
		count++
		_, _ = fmt.Fprintf(ui.Err, "[%d/%d] Imported %s\n", count, len(docs), docMeta.Title)
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully imported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}
