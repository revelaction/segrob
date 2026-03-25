package main

import (
	"errors"
	"fmt"

	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func corpusExportTopicCommand(opts CorpusExportTopicOptions, ui UI) (err error) {
	pool, err := zombiezen.NewPool(opts.DbPath)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, pool.Close())
	}()
	src := zombiezen.NewCorpusTopicStore(pool)

	dst := filesystem.NewTopicStore(opts.Directory)

	topics, err := src.ReadAll()
	if err != nil {
		return err
	}

	for _, tp := range topics {
		if err := dst.Write(tp); err != nil {
			return fmt.Errorf("failed to export topic %s: %w", tp.Name, err)
		}
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully exported %d topics from %s to %s\n", len(topics), opts.DbPath, opts.Directory)
	return nil
}
