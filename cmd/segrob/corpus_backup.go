package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/revelaction/segrob/storage"
)

// corpusBackupCommand populates the destination backup database using source interfaces.
// It expects the dstMgr to have initialized the schema.
func corpusBackupCommand(
	srcRepo storage.CorpusReader,
	srcTopics storage.TopicReader,
	dstMgr storage.SchemaManager,
	dstRepo storage.CorpusWriter,
	dstTopics storage.TopicWriter,
	tempPath string,
	opts CorpusBackupOptions,
	ui UI,
) (err error) {
	defer func() {
		err = errors.Join(err, os.Remove(tempPath))
	}()

	err = dstMgr.Create("corpus.sql")
	if err != nil {
		return fmt.Errorf("failed to create backup schemas: %w", err)
	}

	// List source corpus and copy rows via WriteStream + backupIterator
	metas, lErr := srcRepo.List()
	if lErr != nil {
		return fmt.Errorf("failed to list corpus: %w", lErr)
	}

	_, pErr := fmt.Fprintf(ui.Err, "Backing up %d document(s)...\n", len(metas))
	if pErr != nil {
		return pErr
	}

	seq := backupIterator(srcRepo, metas, opts.WithNlp)
	err = dstRepo.WriteStream(seq)
	if err != nil {
		return err
	}

	// Copy topics
	topics, rErr := srcTopics.ReadAll()
	if rErr != nil {
		return fmt.Errorf("failed to read topics: %w", rErr)
	}

	for _, tp := range topics {
		wErr := dstTopics.Write(tp)
		if wErr != nil {
			return fmt.Errorf("failed to write topic %s: %w", tp.Name, wErr)
		}
	}

	if len(topics) > 0 {
		_, pErr := fmt.Fprintf(ui.Err, "  ✅ %d topic(s)\n", len(topics))
		if pErr != nil {
			return pErr
		}
	}

	var outputPath string
	if opts.Output != "" {
		outputPath = opts.Output
	} else {
		timestamp := time.Now().UTC().Format("20060102T150405Z")
		base := filepath.Base(opts.DbPath)
		outputPath = fmt.Sprintf("%s-%s.gz", base, timestamp)
	}

	err = compressFile(tempPath, outputPath)
	if err != nil {
		removeErr := os.Remove(outputPath)
		return errors.Join(fmt.Errorf("failed to compress backup: %w", err), removeErr)
	}

	_, pErr = fmt.Fprintf(ui.Err, "Backup written to %s\n", outputPath)
	if pErr != nil {
		return pErr
	}
	return nil
}

// backupIterator returns an iter.Seq2 that yields CorpusRecord values
// for each meta. It reads the txt field from the source repo and combines
// it with the metadata. On error, it yields the error and halts.
func backupIterator(srcRepo storage.CorpusReader, metas []storage.CorpusMeta, withNlp bool) func(yield func(storage.CorpusRecord, error) bool) {
	return func(yield func(storage.CorpusRecord, error) bool) {
		for _, m := range metas {
			txt, err := srcRepo.ReadTxt(m.ID)
			if err != nil {
				yield(storage.CorpusRecord{}, fmt.Errorf("failed to read txt for %s: %w", m.ID, err))
				return
			}

			record := storage.CorpusRecord{CorpusMeta: m, Txt: string(txt)}

			if withNlp {
				nlp, err := srcRepo.ReadNlp(m.ID)
				if err != nil {
					yield(storage.CorpusRecord{}, fmt.Errorf("failed to read nlp for %s: %w", m.ID, err))
					return
				}
				record.Nlp = string(nlp)
			}

			if !yield(record, nil) {
				return
			}
		}
	}
}

// compressFile reads source and writes gzipped content to dest.
func compressFile(sourcePath, destPath string) (err error) {
	sourceFile, oerr := os.Open(sourcePath)
	if oerr != nil {
		return fmt.Errorf("failed to open source file: %w", oerr)
	}
	defer func() {
		err = errors.Join(err, sourceFile.Close())
	}()

	destFile, cerr := os.Create(destPath)
	if cerr != nil {
		return fmt.Errorf("failed to create destination file: %w", cerr)
	}
	defer func() {
		err = errors.Join(err, destFile.Close())
	}()

	gzipWriter := gzip.NewWriter(destFile)
	defer func() {
		err = errors.Join(err, gzipWriter.Close())
	}()

	_, cpErr := io.Copy(gzipWriter, sourceFile)
	if cpErr != nil {
		return fmt.Errorf("failed to compress data: %w", cpErr)
	}

	return nil
}
