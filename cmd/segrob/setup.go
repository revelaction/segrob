package main

import (
	"path/filepath"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Setup struct {
	pools map[string]*sqlitex.Pool
}

func NewSetup() *Setup {
	return &Setup{
		pools: make(map[string]*sqlitex.Pool),
	}
}

// NewSchemaManager returns a manager for the database at the given path.
func (s *Setup) NewSchemaManager(path string, params ...string) (storage.SchemaManager, error) {
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewSchemaManager(pool), nil
}

// getPool returns an existing pool for the given path if available, or opens a new one.
// It uses absolute paths to identify unique databases.
func (s *Setup) getPool(path string, params ...string) (*sqlitex.Pool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if pool, ok := s.pools[absPath]; ok {
		return pool, nil
	}

	pool, err := zombiezen.NewPool(absPath, params...)
	if err != nil {
		return nil, err
	}

	s.pools[absPath] = pool
	return pool, nil
}

func (s *Setup) NewDocRepository(path string, params ...string) (storage.DocRepository, error) {
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewDocStore(pool), nil
}

func (s *Setup) NewCorpusRepository(path string, params ...string) (storage.CorpusRepository, error) {
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewCorpusStore(pool), nil
}

func (s *Setup) NewLiveTopicRepository(path string, params ...string) (storage.TopicRepository, error) {
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewLiveTopicStore(pool), nil
}

func (s *Setup) NewCorpusTopicRepository(path string, params ...string) (storage.TopicRepository, error) {
	// Corpus uses sqlite exclusively
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewCorpusTopicStore(pool), nil
}

func (s *Setup) NewLiveTopicDeleter(path string, params ...string) (storage.TopicDeleter, error) {
	// Live commands only use SQLite backend
	pool, err := s.getPool(path, params...)
	if err != nil {
		return nil, err
	}
	return zombiezen.NewLiveTopicStore(pool), nil
}

// Close closes all managed pools.
func (s *Setup) Close() error {
	var firstErr error
	for _, pool := range s.pools {
		if err := pool.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
