package file

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	sent "github.com/revelaction/segrob/sentence"
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
