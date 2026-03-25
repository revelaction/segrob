package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
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
) error {
	err := dstMgr.Create("corpus.sql")
	if err != nil {
		return fmt.Errorf("failed to create backup schemas: %w", err)
	}

	// List source corpus and copy rows via WriteStream + backupIterator
	metas, err := srcRepo.List()
	if err != nil {
		return fmt.Errorf("failed to list corpus: %w", err)
	}

	if _, err := fmt.Fprintf(ui.Err, "Backing up %d document(s)...\n", len(metas)); err != nil {
		return err
	}

	seq := backupIterator(srcRepo, metas, opts.WithNlp)
	err = dstRepo.WriteStream(seq)
	if err != nil {
		return err
	}

	// Copy topics
	topics, err := srcTopics.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read topics: %w", err)
	}

	for _, tp := range topics {
		err := dstTopics.Write(tp)
		if err != nil {
			return fmt.Errorf("failed to write topic %s: %w", tp.Name, err)
		}
	}

	if len(topics) > 0 {
		if _, err := fmt.Fprintf(ui.Err, "  ✅ %d topic(s)\n", len(topics)); err != nil {
			return err
		}
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	outputPath := fmt.Sprintf("%s-%s.db.gz", opts.Output, timestamp)

	err = compressFile(tempPath, outputPath)
	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to compress backup: %w", err)
	}

	fmt.Fprintf(ui.Err, "Backup written to %s\n", outputPath)
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
func compressFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	_, err = io.Copy(gzipWriter, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	return nil
}
