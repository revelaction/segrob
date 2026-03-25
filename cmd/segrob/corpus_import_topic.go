package main

import (
	"errors"
	"fmt"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func corpusImportTopicCommand(opts CorpusImportTopicOptions, ui UI) (err error) {

	src := filesystem.NewTopicStore(opts.Directory)

	pool, err := zombiezen.NewPool(opts.DbPath)

	if err != nil {
		return err
	}

	defer func() {
		err = errors.Join(err, pool.Close())
	}()

	dst := zombiezen.NewCorpusTopicStore(pool)

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
