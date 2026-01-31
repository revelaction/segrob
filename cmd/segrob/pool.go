package main

import (
	"github.com/revelaction/segrob/storage/sqlite/zombiezen"
	"zombiezen.com/go/sqlite/sqlitex"
)

type Pool struct {
	p *sqlitex.Pool
}

func (p *Pool) Open(path string) (*sqlitex.Pool, error) {
	if p.p != nil {
		return p.p, nil
	}
	pool, err := zombiezen.NewPool(path)
	if err != nil {
		return nil, err
	}
	p.p = pool
	return p.p, nil
}

func (p *Pool) Close() error {
	if p.p != nil {
		return p.p.Close()
	}
	return nil
}
