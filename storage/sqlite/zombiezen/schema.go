package zombiezen

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const schemaTopics = `
CREATE TABLE IF NOT EXISTS topics (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id   TEXT,
    name      TEXT NOT NULL,
    exprs     TEXT NOT NULL,
    created   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(user_id, name)
);`

const schemaIndex = `CREATE INDEX IF NOT EXISTS idx_topics_user_id ON topics(user_id);`

func InitSchema(conn *sqlite.Conn) error {
	if err := sqlitex.ExecuteTransient(conn, schemaTopics, nil); err != nil {
		return err
	}
	return sqlitex.ExecuteTransient(conn, schemaIndex, nil)
}
