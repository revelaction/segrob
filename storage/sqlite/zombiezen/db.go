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

	// Use DSN parameters: 
	params := []string{
		"_journal_mode=WAL",   // WAL mode is the critical enabler: readers never block writers, and writers never block readers.
		"_synchronous=NORMAL", // Recommended mode for WAL; provides better performance.
		"_busy_timeout=200",   // Wait up to 200ms for a write lock before returning SQLITE_BUSY; in ms
		"_foreign_keys=on",    // Enforce foreign key constraints; otherwise they are decorative.
		"_cache_size=-64000",  // Set page cache size (2MB).
        "_txlock=immediate"    // Acquire write lock immediately, don't wait until first write. This prevents the common pattern where transactions start optimistically and then fail when they try to write.
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
