package main

import (
	"path/filepath"

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

// GetPool returns an existing pool for the given path if available, or opens a new one.
// It uses absolute paths to identify unique databases.
func (s *Setup) GetPool(path string) (*sqlitex.Pool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if pool, ok := s.pools[absPath]; ok {
		return pool, nil
	}

	pool, err := zombiezen.NewPool(absPath)
	if err != nil {
		return nil, err
	}

	s.pools[absPath] = pool
	return pool, nil
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
