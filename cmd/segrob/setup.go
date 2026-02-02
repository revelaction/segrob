package main

import (
	"fmt"
	"os"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/filesystem"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
)

func NewTopicRepository(p *Pool, path string) (storage.TopicRepository, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %s", path)
	}

	if info.IsDir() {
		return filesystem.NewTopicStore(path), nil
	}

	pool, err := p.Open(path)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewTopicStore(pool), nil
}

func NewDocRepository(p *Pool, path string) (storage.DocRepository, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("repository not found: %s", path)
	}

	if info.IsDir() {
		return filesystem.NewDocStore(path)
	}

	pool, err := p.Open(path)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewDocStore(pool), nil
}