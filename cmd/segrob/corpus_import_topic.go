package main

import (
	"fmt"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
)

func corpusImportTopicCommand(dst storage.TopicWriter, opts CorpusImportTopicOptions, ui UI) error {

	src := filesystem.NewTopicStore(opts.Directory)

	topics, err := src.ReadAll()

	if err != nil {
		return err
	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to import topic %s: %w", tp.Name, err)
		}
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully imported %d topics from %s to %s\n", len(topics), opts.Directory, opts.DbPath)
	return nil
}
