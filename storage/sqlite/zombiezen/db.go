package zombiezen

import (
	"fmt"
	"runtime"
	"strings"

	"zombiezen.com/go/sqlite/sqlitex"
)

// NewPool creates a new Zombiezen SQLite connection pool with reasonable defaults
// (e.g., WAL mode enabled).
func NewPool(dbPath string) (*sqlitex.Pool, error) {
	poolSize := runtime.NumCPU()

	// Construct the DSN string with performance PRAGMAs
	// Use DSN parameters: _journal_mode, _synchronous, _busy_timeout, _foreign_keys, _cache_size
	// busy_timeout in DSN is in milliseconds.
	params := []string{
		"_journal_mode=WAL",   // WAL mode is the critical enabler: readers never block writers, and writers never block readers.
		"_synchronous=NORMAL", // Recommended mode for WAL; provides better performance.
		"_busy_timeout=200",   // Wait up to 200ms for a write lock before returning SQLITE_BUSY; essential for multi-writer scenarios.
		"_foreign_keys=on",    // Enforce foreign key constraints; otherwise they are decorative.
		"_cache_size=-2000",   // Set page cache size (2MB).
	}
	initString := fmt.Sprintf("file:%s?%s", dbPath, strings.Join(params, "&"))

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
