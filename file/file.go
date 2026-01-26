package file

import (
	"bytes"
	"encoding/json"
	sent "github.com/revelaction/segrob/sentence"
	tpc "github.com/revelaction/segrob/topic"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

const (
	TokenDir = "./corpus/token/"
	TopicDir = "./corpus/topic/"
)

type DocHandler struct{}

func NewDocHandler() (*DocHandler, error) {
	return &DocHandler{}, nil
}

// ReadDoc reads a Doc JSON from the given path and unmarshals it.
func ReadDoc(path string) (sent.Doc, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return sent.Doc{}, err
	}

	var doc sent.Doc
	err = json.Unmarshal(f, &doc)
	if err != nil {
		return sent.Doc{}, err
	}

	return doc, nil
}

func (h *DocHandler) Names() ([]string, error) {
	f, err := files(TokenDir)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (h *DocHandler) DocForName(name string) (sent.Doc, error) {
	f, err := ioutil.ReadFile(TokenDir + name)
	if err != nil {
		return sent.Doc{}, err
	}

	var doc sent.Doc
	err = json.Unmarshal(f, &doc)
	if err != nil {
		return sent.Doc{}, err
	}

	doc.Title = name
	return doc, nil
}

func (h *DocHandler) Random() (sent.Doc, error) {
	all, err := h.Names()
	if err != nil {
		return sent.Doc{}, err
	}

	rand.Seed(time.Now().Unix())
	idx := rand.Intn(len(all))
	return h.DocForName(all[idx])
}

func (h *DocHandler) Doc(docId int) (sent.Doc, error) {
	names, err := h.Names()
	if err != nil {
		return sent.Doc{}, err
	}

	return h.DocForName(names[docId])
}

func (h *DocHandler) DocForMatch(match string) (sent.Doc, error) {
	names, err := h.Names()
	if err != nil {
		return sent.Doc{}, err
	}

	for _, fname := range names {
		if strings.Contains(fname, match) {
			return h.DocForName(fname)
		}
	}

	return sent.Doc{}, err
}

func files(dir string) ([]string, error) {
	txtFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fileNames := []string{}
	for _, file := range txtFiles {
		if filepath.Ext(file.Name()) == ".json" {
			fileNames = append(fileNames, file.Name())
		}
	}

	return fileNames, nil
}

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
