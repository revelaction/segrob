package main

import (
	"fmt"
	"os"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func catTxtCommand(opts CatTxtOptions, ui UI) error {
	pool, err := zombiezen.NewPool(opts.DbPath)
	if err != nil {
		return fmt.Errorf("failed to open corpus database: %w", err)
	}
	defer pool.Close()

	store := zombiezen.NewCorpusStore(pool)

	txt, err := store.ReadTxt(opts.ID)
	if err != nil {
		return fmt.Errorf("failed to read txt for %s: %w", opts.ID, err)
	}

	// Write byte-exact to file or stdout
	if opts.Output != "" {
		if err := os.WriteFile(opts.Output, txt, 0644); err != nil {
			return fmt.Errorf("failed to write output file %s: %w", opts.Output, err)
		}
		return nil
	}

	// stdout: write raw bytes directly
	_, err = ui.Out.Write(txt)
	return err
}
