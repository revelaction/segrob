package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

// sha256Hex returns the hex-encoded SHA-256 of data, truncated to n bytes.
func sha256Hex(data []byte, n int) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:n])
}

func corpusCommand(opts CorpusOptions, ui UI) error {
	// Check pandoc exists
	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf("pandoc is not installed or not in PATH: %w", err)
	}

	// Open/create the corpus database
	pool, err := zombiezen.NewPool(opts.OutputDb)
	if err != nil {
		return fmt.Errorf("failed to open corpus database: %w", err)
	}
	defer pool.Close()

	// Create schema if not exists
	if err := zombiezen.CreateSchemas(pool, "corpus.sql"); err != nil {
		return fmt.Errorf("failed to create corpus schema: %w", err)
	}

	store := zombiezen.NewCorpusStore(pool)

	// Collect epub file paths from the single directory (flat, no recursion)
	var epubPaths []string
	entries, err := os.ReadDir(opts.Dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", opts.Dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".epub" {
			epubPaths = append(epubPaths, filepath.Join(opts.Dir, entry.Name()))
		}
	}

	if len(epubPaths) == 0 {
		fmt.Fprintf(ui.Err, "No epub files found in %s.\n", opts.Dir)
		return nil
	}

	// Build iterator and write stream
	seq := corpusIterator(store, epubPaths, ui)
	if err := store.WriteStream(seq); err != nil {
		return err
	}

	return nil
}
