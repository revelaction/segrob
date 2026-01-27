package file

import (
	"encoding/json"
	"os"

	sent "github.com/revelaction/segrob/sentence"
)

const (
	TokenDir = "./corpus/token/"
	TopicDir = "./corpus/topic/"
)

// ReadDoc reads a Doc JSON from the given path and unmarshals it.
func ReadDoc(path string) (sent.Doc, error) {
	f, err := os.ReadFile(path)
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
