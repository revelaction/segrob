package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/revelaction/segrob/storage"
)

func corpusDumpTopicCommand(src storage.TopicReader, opts CorpusDumpTopicOptions, name string, ui UI) error {
	tp, err := src.Read(name)
	if err != nil {
		return fmt.Errorf("failed to read topic %s: %w", name, err)
	}

	jsonData, err := json.Marshal(tp.Exprs)
	if err != nil {
		return fmt.Errorf("failed to marshal topic %s: %w", name, err)
	}

	// Format the json with each line containing a topic expression
	jsonFmt := bytes.TrimPrefix(jsonData, []byte("["))
	jsonFmt = bytes.ReplaceAll(jsonFmt, []byte("],"), []byte("],\n\t"))
	jsonFmt = bytes.TrimSuffix(jsonFmt, []byte("]"))
	jsonFmt = append([]byte("[\n\t"), jsonFmt...)
	jsonFmt = append(jsonFmt, []byte("\n]")...)

	_, err = ui.Out.Write(jsonFmt)
	return err
}
