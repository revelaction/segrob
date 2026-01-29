package zombiezen

import (
	"fmt"
	"runtime"

	"zombiezen.com/go/sqlite/sqlitex"
)

// NewPool creates a new Zombiezen SQLite connection pool with reasonable defaults
// (e.g., WAL mode enabled).
func NewPool(dbPath string) (*sqlitex.Pool, error) {
	poolSize := runtime.NumCPU()
	initString := fmt.Sprintf("file:%s", dbPath)

	// zombiezen/sqlitex.NewPool with default options uses flags:
	// sqlite.OpenReadWrite | sqlite.OpenCreate | sqlite.OpenWAL | sqlite.OpenURI
	pool, err := sqlitex.NewPool(initString, sqlitex.PoolOptions{
		PoolSize: poolSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create default zombiezen pool at %s: %w", dbPath, err)
	}
	return pool, nil
}
