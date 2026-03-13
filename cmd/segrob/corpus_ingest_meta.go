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
// Hex chars  Bytes   Bits   Collision probability 
// 16         8       64     ~0.000000000003% 😅 overkill
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

func openBook(epubBytes []byte, name string) (*epub.Book, error) {
	// IMPORTANT NOTE: zr is created from an in-memory byte slice using bytes.NewReader.
	// Unlike zip.OpenReader which returns a *zip.ReadCloser (requiring explicit defer rc.Close()),
	// zip.NewReader returns a *zip.Reader that does not hold file handles and has no Close() method.
	// The resource remains safely alive/open for lazy reading while `book` is in scope.
	zr, err := zip.NewReader(bytes.NewReader(epubBytes), int64(len(epubBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to open epub zip %s: %w", name, err)
	}
	book, err := epub.New(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse epub %s: %w", name, err)
	}
	return book, nil
}

func finalizeText(record *storage.CorpusRecord, rawTxt string) {
	record.Txt = epub.CleanForNLP(rawTxt)
	record.TxtHash = sha256Hex([]byte(record.Txt), 32)
}

func corpusIngestMetaCommand(pool *sqlitex.Pool, repo storage.CorpusRepository, opts CorpusIngestMetaOptions, ui UI) error {
	// Select processor once here. Nothing below this point knows about the flag.
	process := processEpubGo
	if opts.Pandoc {
		if _, err := exec.LookPath("pandoc"); err != nil {
			return fmt.Errorf("pandoc is not installed or not in PATH: %w", err)
		}
		process = processEpubPandoc
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
	seq := corpusIterator(repo, paths, process, ui)
	if err := repo.WriteStream(seq); err != nil {
		return err
	}

	return nil
}

func processEpubGo(epubBytes []byte, name, id string) (storage.CorpusRecord, error) {
	record := storage.CorpusRecord{CorpusMeta: storage.CorpusMeta{ID: id, Epub: name}}
	book, err := openBook(epubBytes, name)
	if err != nil {
		return record, err
	}
	record.Labels = book.Labels()
	txt, err := book.Text()
	if err != nil {
		return record, fmt.Errorf("failed to extract text from %s: %w", name, err)
	}
	finalizeText(&record, txt)
	return record, nil
}

func processEpubPandoc(epubBytes []byte, name, id string) (storage.CorpusRecord, error) {
	record := storage.CorpusRecord{CorpusMeta: storage.CorpusMeta{ID: id, Epub: name}}
	book, err := openBook(epubBytes, name)
	if err != nil {
		return record, err
	}
	record.Labels = book.Labels()
	cmd := exec.Command("pandoc", "-f", "epub", "-t", "plain", "--wrap=preserve")
	cmd.Stdin = bytes.NewReader(epubBytes)
	out, err := cmd.Output()
	if err != nil {
		return record, fmt.Errorf("pandoc failed for %s: %w", name, err)
	}
	finalizeText(&record, string(out))
	return record, nil
}

// corpusIterator returns an iter.Seq2 that yields CorpusRecord values for
// each epub path. It checks existence in the store (idempotency) and
// prints a summary line per processed epub. On error, it yields the error
// and halts.
func corpusIterator(repo storage.CorpusRepository, epubPaths []string, process func([]byte, string, string) (storage.CorpusRecord, error), ui UI) func(yield func(storage.CorpusRecord, error) bool) {
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
			// 16 hex chars
			id := sha256Hex(epubBytes, 8)

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
			record, err := process(epubBytes, name, id)
			if err != nil {
				yield(storage.CorpusRecord{}, err)
				return
			}

			// Print summary line: labels and text length in UTF-8 characters
			charCount := utf8.RuneCountInString(record.Txt)
			fmt.Fprintf(ui.Err, "%s | labels: %s | chars: %d\n", record.Epub, record.Labels, charCount)

			if !yield(record, nil) {
				return
			}
		}
	}
}
