package main

import (
	"fmt"

	"github.com/gosuri/uiprogress"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func importDocCommand(opts ImportDocOptions, ui UI) error {
	src, err := filesystem.NewDocHandler(opts.From)
	if err != nil {
		return err
	}
	if err := src.LoadList(); err != nil {
		return err
	}
	if err := src.LoadContents(nil); err != nil {
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

	fmt.Fprintf(ui.Out, "Reading docs from %s...\n", opts.From)
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

		if err := dst.WriteDoc(doc); err != nil {
			uiprogress.Stop()
			return fmt.Errorf("failed to write doc %s: %w", docMeta.Title, err)
		}
		count++
		bar.Incr()
	}
	uiprogress.Stop()

	fmt.Fprintf(ui.Out, "Successfully imported %d docs from %s to %s\n", count, opts.From, opts.To)
	return nil
}
