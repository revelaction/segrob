package filesystem

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/revelaction/segrob/storage"
	tpc "github.com/revelaction/segrob/topic"
)

type TopicStore struct {
	root string
}

var _ storage.TopicReader = (*TopicStore)(nil)
var _ storage.TopicWriter = (*TopicStore)(nil)

func NewTopicStore(root string) *TopicStore {
	return &TopicStore{root: root}
}

func (th *TopicStore) ReadAll() (tpc.Library, error) {
	names, err := th.names()
	if err != nil {
		return nil, err
	}

	topics := tpc.Library{}
	for _, n := range names {
		t, err := th.Read(n)
		if err != nil {
			return nil, err
		}

		topics = append(topics, t)
	}

	return topics, nil
}

func (th *TopicStore) names() ([]string, error) {
	files, err := os.ReadDir(th.root)
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

func (th *TopicStore) Read(name string) (tpc.Topic, error) {
	tf, err := os.ReadFile(filepath.Join(th.root, name+".json"))
	if err != nil {
		return tpc.Topic{}, err
	}

	var t tpc.Topic
	t.Name = name

	exprs := []tpc.TopicExpr{}
	err = json.Unmarshal(tf, &exprs)
	if err != nil {
		return tpc.Topic{}, err
	}

	// Set the topic name and expresion Id in each Item
	for index := range exprs {
		for idx := range exprs[index] {
			exprs[index][idx].TopicName = name
			exprs[index][idx].ExprIndex = index
			exprs[index][idx].ItemIndex = idx
		}
	}

	t.Exprs = exprs
	return t, nil
}

func (th *TopicStore) Write(tp tpc.Topic) error {
	jsonData, err := json.Marshal(tp.Exprs)
	if err != nil {
		return err
	}

	// Format the json with each line containing a topic expresion
	// Remove the first [
	jsonFmt := bytes.TrimPrefix(jsonData, []byte("["))
	// replace the rest with indent
	jsonFmt = bytes.ReplaceAll(jsonFmt, []byte("],"), []byte("],\n\t"))
	// remove the last
	jsonFmt = bytes.TrimSuffix(jsonFmt, []byte("]"))
	jsonFmt = append([]byte("[\n\t"), jsonFmt...)
	jsonFmt = append(jsonFmt, []byte("\n]")...)

	err = os.WriteFile(filepath.Join(th.root, tp.Name+".json"), jsonFmt, 0644)
	if err != nil {
		return err
	}

	return nil
}
