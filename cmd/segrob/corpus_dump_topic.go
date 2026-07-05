package main

import (
	"encoding/json"
	"fmt"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

func corpusDumpTopicCommand(src storage.TopicReader, opts CorpusDumpTopicOptions, name string, ui UI) error {
	tp, err := src.Read("", name)
	if err != nil {
		return fmt.Errorf("failed to read topic %s: %w", name, err)
	}

	tp.Exprs = tpc.Deduplicate(tp.Exprs)

	jsonData, err := json.MarshalIndent(tp.Exprs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal topic %s: %w", name, err)
	}

	_, err = ui.Out.Write(jsonData)
	return err
}
