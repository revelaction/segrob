package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

// topicFileNames scans dir for *.json files and returns their basenames without extension.
func topicFileNames(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	names := []string{}
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		names = append(names, strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())))
	}

	return names, nil
}

// readTopicFile reads a single <name>.json from dir and returns the parsed Topic.
func readTopicFile(dir, name string) (tpc.Topic, error) {
	tf, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return tpc.Topic{}, err
	}

	exprs := []tpc.TopicExpr{}
	err = json.Unmarshal(tf, &exprs)
	if err != nil {
		return tpc.Topic{}, err
	}

	// Set the topic name and expression Id in each Item
	for index := range exprs {
		for idx := range exprs[index] {
			exprs[index][idx].TopicName = name
			exprs[index][idx].ExprIndex = index
			exprs[index][idx].ItemIndex = idx
		}
	}

	return tpc.Topic{Name: name, Exprs: exprs}, nil
}

// readTopicFiles scans dir for *.json topic files and returns all parsed topics.
func readTopicFiles(dir string) (tpc.Library, error) {
	names, err := topicFileNames(dir)
	if err != nil {
		return nil, err
	}

	topics := tpc.Library{}
	for _, n := range names {
		t, err := readTopicFile(dir, n)
		if err != nil {
			return nil, err
		}

		topics = append(topics, t)
	}

	return topics, nil
}

func corpusIngestTopicCommand(dst storage.TopicWriter, opts CorpusIngestTopicOptions, ui UI) error {
	topics, err := readTopicFiles(opts.Dir)
	if err != nil {
		return err
	}

	if len(topics) == 0 {
		_, _ = fmt.Fprintf(ui.Err, "No JSON topic files found in %s.\n", opts.Dir)
		return nil
	}

	for _, tp := range topics {
		err = dst.Write(tp)
		if err != nil {
			return fmt.Errorf("failed to ingest topic %s: %w", tp.Name, err)
		}
	}

	_, _ = fmt.Fprintf(ui.Err, "Successfully ingested %d topics from %s\n", len(topics), opts.Dir)
	return nil
}
