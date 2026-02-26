package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/revelaction/segrob/epub"
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

// processEpub reads epub bytes from r, computes the hash, extracts DC labels,
// runs pandoc (via stdin) to convert it to plain text, and returns a CorpusRecord.
func processEpub(r io.Reader, name string) (zombiezen.CorpusRecord, error) {
	var record zombiezen.CorpusRecord

	epubBytes, err := io.ReadAll(r)
	if err != nil {
		return record, fmt.Errorf("failed to read epub %s: %w", name, err)
	}

	// Compute epub hash for ID (truncated to 16 bytes = 32 hex chars)
	record.ID = sha256Hex(epubBytes, 16)
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
