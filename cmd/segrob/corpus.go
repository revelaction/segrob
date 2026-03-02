package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"unicode/utf8"

	"github.com/revelaction/segrob/epub"
	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"zombiezen.com/go/sqlite/sqlitex"
)

// sha256Hex returns the hex-encoded SHA-256 of data, truncated to n bytes.
func sha256Hex(data []byte, n int) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:n])
}

// epubPaths returns a list of absolute paths to epub files in the given directory.
// It is a flat scan (no recursion).
func epubPaths(dir string) ([]string, error) {
	var paths []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".epub" {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return paths, nil
}

func corpusCommand(pool *sqlitex.Pool, repo storage.CorpusRepository, opts CorpusOptions, ui UI) error {
	// Check pandoc exists
	if _, err := exec.LookPath("pandoc"); err != nil {
		return fmt.Errorf("pandoc is not installed or not in PATH: %w", err)
	}

	// Create schema if not exists
	if err := zombiezen.CreateSchemas(pool, "corpus.sql"); err != nil {
		return fmt.Errorf("failed to create corpus schema: %w", err)
	}

	// Collect epub file paths from the single directory (flat, no recursion)
	paths, err := epubPaths(opts.Dir)
	if err != nil {
		return err
	}

	if len(paths) == 0 {
		fmt.Fprintf(ui.Err, "No epub files found in %s.\n", opts.Dir)
		return nil
	}

	// Build iterator and write stream
	seq := corpusIterator(repo, paths, ui)
	if err := repo.WriteStream(seq); err != nil {
		return err
	}

	return nil
}

// processEpub takes pre-read epub bytes and a pre-computed id, extracts DC labels,
// runs pandoc (via stdin) to convert it to plain text, and returns a CorpusRecord.
func processEpub(epubBytes []byte, name, id string) (storage.CorpusRecord, error) {
	var record storage.CorpusRecord
	record.ID = id
	record.Epub = name

	// Extract DC labels via *zip.Reader (decoupled from filesystem)
	zr, err := zip.NewReader(bytes.NewReader(epubBytes), int64(len(epubBytes)))
	if err != nil {
		return record, fmt.Errorf("failed to open epub zip %s: %w", name, err)
	}

	labels, err := epub.Labels(zr)
	if err != nil {
		return record, fmt.Errorf("failed to extract epub labels from %s: %w", name, err)
	}
	record.Labels = labels

	// Run pandoc to convert epub to plain text via stdin
	cmd := exec.Command("pandoc", "-f", "epub", "-t", "plain", "--wrap=preserve")
	cmd.Stdin = bytes.NewReader(epubBytes)
	txtBytes, err := cmd.Output()
	if err != nil {
		return record, fmt.Errorf("pandoc failed for %s: %w", name, err)
	}

	record.Txt = string(txtBytes)

	// Hash the full text output (full 32 bytes = 64 hex chars)
	record.TxtHash = sha256Hex(txtBytes, 32)

	return record, nil
}

// corpusIterator returns an iter.Seq2 that yields CorpusRecord values for
// each epub path. It checks existence in the store (idempotency) and
// prints a summary line per processed epub. On error, it yields the error
// and halts.
func corpusIterator(repo storage.CorpusRepository, epubPaths []string, ui UI) func(yield func(storage.CorpusRecord, error) bool) {
	seen := make(map[string]bool)
	return func(yield func(storage.CorpusRecord, error) bool) {
		for _, epubPath := range epubPaths {
			name := filepath.Base(epubPath)

			epubBytes, err := os.ReadFile(epubPath)
			if err != nil {
				yield(storage.CorpusRecord{}, fmt.Errorf("failed to read %s: %w", epubPath, err))
				return
			}

			// Compute hash early to skip duplicates before heavy work
			id := sha256Hex(epubBytes, 16)

			if seen[id] {
				fmt.Fprintf(ui.Err, "[skip] %s (duplicate in batch)\n", name)
				continue
			}

			exists, err := repo.Exists(id)
			if err != nil {
				yield(storage.CorpusRecord{}, fmt.Errorf("failed to check existence for %s: %w", epubPath, err))
				return
			}
			if exists {
				fmt.Fprintf(ui.Err, "[skip] %s (already in corpus)\n", name)
				continue
			}

			seen[id] = true

			// Now do the expensive work: zip parsing, label extraction, pandoc
			record, err := processEpub(epubBytes, name, id)
			if err != nil {
				yield(storage.CorpusRecord{}, err)
				return
			}

			// Print summary line: labels and text length in UTF-8 characters
			charCount := utf8.RuneCountInString(record.Txt)
			fmt.Fprintf(ui.Out, "%s | labels: %s | chars: %d\n", record.Epub, record.Labels, charCount)

			if !yield(record, nil) {
				return
			}
		}
	}
}
