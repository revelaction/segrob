package zombiezen

import (
	"context"
	"fmt"

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

const schemaDocs = `
CREATE TABLE IF NOT EXISTS docs (
    id      INTEGER PRIMARY KEY,
    title   TEXT NOT NULL UNIQUE,
    labels  TEXT
);`

const schemaSentences = `
CREATE TABLE IF NOT EXISTS sentences (
    rowid   INTEGER PRIMARY KEY,  -- Implicit, used for fast pagination
    doc_id  INTEGER NOT NULL,
    data    TEXT NOT NULL,        -- Full JSON []sent.Token
    FOREIGN KEY (doc_id) REFERENCES docs(id)
);`

const schemaSentenceLemmas = `
CREATE TABLE IF NOT EXISTS sentence_lemmas (
    lemma           TEXT NOT NULL,
    sentence_rowid  INTEGER NOT NULL,
    FOREIGN KEY (sentence_rowid) REFERENCES sentences(rowid)
);`

const schemaLemmaIndex = `
CREATE INDEX IF NOT EXISTS idx_lemma_rowid ON sentence_lemmas(lemma, sentence_rowid);`

const schemaDocsSentencesIndex = `
CREATE INDEX IF NOT EXISTS idx_sentences_doc_id ON sentences(doc_id);`

func CreateTopicTables(pool *sqlitex.Pool) error {
	conn, err := pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	if err := sqlitex.ExecuteTransient(conn, schemaTopics, nil); err != nil {
		return fmt.Errorf("failed to create topics table: %w", err)
	}
	if err := sqlitex.ExecuteTransient(conn, schemaIndex, nil); err != nil {
		return fmt.Errorf("failed to create user_id index: %w", err)
	}
	return nil
}

func CreateDocTables(pool *sqlitex.Pool) error {
	conn, err := pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer pool.Put(conn)

	schemas := []string{
		schemaDocs,
		schemaSentences,
		schemaSentenceLemmas,
		schemaLemmaIndex,
		schemaDocsSentencesIndex,
	}

	for _, schema := range schemas {
		if err := sqlitex.ExecuteTransient(conn, schema, nil); err != nil {
			return fmt.Errorf("failed to execute schema: %w\n%s", err, schema)
		}
	}

	return nil
}
