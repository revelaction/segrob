package filesystem

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	tpc "github.com/revelaction/segrob/topic"
)

type TopicHandler struct {
	root string
}

var _ tpc.TopicReader = (*TopicHandler)(nil)
var _ tpc.TopicWriter = (*TopicHandler)(nil)

func NewTopicHandler(root string) *TopicHandler {
	return &TopicHandler{root: root}
}

func (th *TopicHandler) All() ([]tpc.Topic, error) {
	names, err := th.Names()
	if err != nil {
		return nil, err
	}

	topics := []tpc.Topic{}
	for _, n := range names {
		t, err := th.Topic(n)
		if err != nil {
			return nil, err
		}

		topics = append(topics, t)
	}

	return topics, nil
}

func (th *TopicHandler) Names() ([]string, error) {
	files, err := ioutil.ReadDir(th.root)
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

func (th *TopicHandler) Random() (tpc.Topic, error) {
	all, err := th.Names()
	if err != nil {
		return tpc.Topic{}, err
	}

	rand.Seed(time.Now().Unix())
	idx := rand.Intn(len(all))
	return th.Topic(all[idx])
}

func (th *TopicHandler) Topic(name string) (tpc.Topic, error) {
	tf, err := ioutil.ReadFile(filepath.Join(th.root, name+".json"))
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
			exprs[index][idx].ExprId = exprs[index].String()
		}
	}

	t.Exprs = exprs
	return t, nil
}

func (th *TopicHandler) Write(tp tpc.Topic) error {
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

	err = ioutil.WriteFile(filepath.Join(th.root, tp.Name+".json"), jsonFmt, 0644)
	if err != nil {
		return err
	}

	return nil
}
