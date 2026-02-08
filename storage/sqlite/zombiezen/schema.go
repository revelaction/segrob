package zombiezen

import (
	"context"
	"embed"
	"fmt"
	"path"

	"zombiezen.com/go/sqlite/sqlitex"
)

// sqlFiles embeds all SQL scripts from the sql/ subdirectory.
//go:embed sql/*.sql
var sqlFiles embed.FS

// CreateSchemas reads a SQL script from the embedded filesystem (e.g., "docs.sql" or "topics.sql")
// and executes it using the provided connection pool.
// It uses context.TODO() for connection acquisition, following local package conventions.
func CreateSchemas(pool *sqlitex.Pool, schemaName string) error {
	// Construct the path within the embedded FS.
	scriptPath := path.Join("sql", schemaName)

	// Call the embed FS to retrieve the script content.
	script, err := sqlFiles.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read embedded sql file %s: %w", scriptPath, err)
	}

	// Acquire a connection from the pool using context.TODO().
	conn, err := pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	// Execute the entire script. ExecuteScript handles multi-statement strings.
	if err := sqlitex.ExecuteScript(conn, string(script), nil); err != nil {
		return fmt.Errorf("failed to execute script %s: %w", schemaName, err)
	}

	return nil
}