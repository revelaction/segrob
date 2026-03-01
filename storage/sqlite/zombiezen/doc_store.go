package zombiezen

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type DocStore struct {
	pool *sqlitex.Pool
}

var _ storage.DocRepository = (*DocStore)(nil)

func NewDocStore(pool *sqlitex.Pool) *DocStore {
	return &DocStore{pool: pool}
}

// upsertLabel inserts a label if it doesn't exist and returns its ID.
// It accepts a sqlite.Conn to allow being called within an existing transaction.
func (h *DocStore) upsertLabel(conn *sqlite.Conn, name string) (int, error) {
	var labelID int
	err := sqlitex.Execute(conn,
		"INSERT INTO labels (name) VALUES (?) ON CONFLICT(name) DO UPDATE SET name = name RETURNING id",
		&sqlitex.ExecOptions{
			Args: []interface{}{name},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				labelID = stmt.ColumnInt(0)
				return nil
			},
		})
	if err != nil {
		return 0, err
	}
	return labelID, nil
}

func (h *DocStore) List() ([]sent.Meta, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var metas []sent.Meta
	err = sqlitex.Execute(conn, "SELECT id, source FROM docs ORDER BY source", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			metas = append(metas, sent.Meta{
				Id:     stmt.ColumnText(0),
				Source: stmt.ColumnText(1),
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return metas, nil
}

func (h *DocStore) Labels(id string) ([]sent.Label, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var labels []sent.Label
	err = sqlitex.Execute(conn, "SELECT l.id, l.name FROM doc_labels dl JOIN labels l ON dl.label_id = l.id WHERE dl.doc_id = ?", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			labels = append(labels, sent.Label{
				ID:   stmt.ColumnInt(0),
				Name: stmt.ColumnText(1),
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return labels, nil
}

func (h *DocStore) Nlp(id string) ([]sent.Sentence, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var sentences []sent.Sentence
	err = sqlitex.Execute(conn, "SELECT sentence_id, data FROM sentences WHERE doc_id = ? ORDER BY sentence_id", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			sentenceID := stmt.ColumnInt(0)
			data := stmt.ColumnText(1)
			var tokens []sent.Token
			if err := json.Unmarshal([]byte(data), &tokens); err != nil {
				return err
			}
			sentences = append(sentences, sent.Sentence{
				SentenceId: sentenceID,
				DocId:      id,
				Tokens:     tokens,
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	return sentences, nil
}

// HasSentences returns true if at least one sentence exists for the given doc ID.
func (h *DocStore) HasSentences(id string) (bool, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return false, err
	}
	defer h.pool.Put(conn)

	var has bool
	// Using the cheapest possible query to check for existence
	err = sqlitex.Execute(conn,
		"SELECT 1 FROM sentences WHERE doc_id = ? LIMIT 1",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				has = true
				return nil
			},
		})
	return has, err
}

func (h *DocStore) FindCandidates(lemmas []string, labelIDs []int, after storage.Cursor, limit int, onCandidate func(sent.Sentence) error) (storage.Cursor, error) {
	if len(lemmas) == 0 {
		return after, nil
	}

	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return after, err
	}
	defer h.pool.Put(conn)

	query, args := h.buildCandidateQuery(lemmas, labelIDs, after, limit)

	// We need to fetch the rowIDs first
	var rowIDs []int64
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			rowIDs = append(rowIDs, stmt.ColumnInt64(0))
			return nil
		},
	})
	if err != nil {
		return after, err
	}

	if len(rowIDs) == 0 {
		return after, nil
	}

	// TODO: Consolidate into a single query using a subquery for better performance.
	// For now, we use a second bulk query to fetch the sentence data.
	idStrings := make([]string, len(rowIDs))
	for i, id := range rowIDs {
		idStrings[i] = strconv.FormatInt(id, 10)
	}
	idList := strings.Join(idStrings, ",")

	bulkQuery := fmt.Sprintf("SELECT rowid, doc_id, sentence_id, data FROM sentences WHERE rowid IN (%s) ORDER BY rowid", idList)

	newCursor := after
	err = sqlitex.Execute(conn, bulkQuery, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			rowID := stmt.ColumnInt64(0)
			if storage.Cursor(rowID) > newCursor {
				newCursor = storage.Cursor(rowID)
			}

			s := sent.Sentence{
				DocId:      stmt.ColumnText(1),
				SentenceId: stmt.ColumnInt(2),
			}
			data := stmt.ColumnText(3)
			if err := json.Unmarshal([]byte(data), &s.Tokens); err != nil {
				return err
			}

			return onCandidate(s)
		},
	})
	if err != nil {
		return after, err
	}

	return newCursor, nil
}

func (h *DocStore) buildCandidateQuery(lemmas []string, labelIDs []int, after storage.Cursor, limit int) (string, []interface{}) {
	var queryBuilder strings.Builder
	var args []interface{}

	// Outer scan: first lemma drives the query
	queryBuilder.WriteString("SELECT sentence_rowid FROM sentence_lemmas AS s_outer WHERE lemma = ? AND sentence_rowid > ?")
	args = append(args, lemmas[0], int(after))

	// Remaining lemmas: EXISTS probes using reverse index (sentence_rowid, lemma)
	for _, lemma := range lemmas[1:] {
		queryBuilder.WriteString(" AND EXISTS (SELECT 1 FROM sentence_lemmas WHERE sentence_rowid = s_outer.sentence_rowid AND lemma = ?)")
		args = append(args, lemma)
	}

	// Labels: EXISTS probes using reverse index (sentence_rowid, label_id)
	for _, labelID := range labelIDs {
		queryBuilder.WriteString(" AND EXISTS (SELECT 1 FROM sentence_labels WHERE sentence_rowid = s_outer.sentence_rowid AND label_id = ?)")
		args = append(args, labelID)
	}

	queryBuilder.WriteString(" ORDER BY sentence_rowid ASC LIMIT ?")
	args = append(args, limit)

	return queryBuilder.String(), args
}

func (h *DocStore) ListLabels(labelSubStr string) ([]sent.Label, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	query, args := h.buildListLabelsQuery(labelSubStr)

	var labels []sent.Label
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			labels = append(labels, sent.Label{
				ID:   stmt.ColumnInt(0),
				Name: stmt.ColumnText(1),
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return labels, nil
}

// buildListLabelsQuery constructs the SQL query for listing labels.
//
// Without labelSubStr:
//
//	SELECT id, name FROM labels ORDER BY name
//
// With labelSubStr:
//
//	SELECT id, name FROM labels WHERE name LIKE ? ORDER BY name
func (h *DocStore) buildListLabelsQuery(labelSubStr string) (string, []interface{}) {
	var queryBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString("SELECT id, name FROM labels")

	if labelSubStr != "" {
		queryBuilder.WriteString(" WHERE name LIKE ?")
		args = append(args, "%"+labelSubStr+"%")
	}

	queryBuilder.WriteString(" ORDER BY name")

	return queryBuilder.String(), args
}

func (h *DocStore) WriteMeta(id string, source string, labels []string) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	err = sqlitex.Execute(conn, "INSERT INTO docs (id, source) VALUES (?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{id, source},
	})
	if err != nil {
		return fmt.Errorf("failed to insert doc: %w", err)
	}

	for _, label := range labels {
		labelID, err := h.upsertLabel(conn, label)
		if err != nil {
			return fmt.Errorf("failed to upsert label %s: %w", label, err)
		}
		err = sqlitex.Execute(conn, "INSERT INTO doc_labels (doc_id, label_id) VALUES (?, ?)", &sqlitex.ExecOptions{
			Args: []interface{}{id, labelID},
		})
		if err != nil {
			return fmt.Errorf("failed to insert doc_label: %w", err)
		}
	}

	return nil
}

func (h *DocStore) WriteNLP(docID string, sentences []storage.SentenceIngest) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	// Fetch existing label IDs for this doc
	var labelIDs []int
	err = sqlitex.Execute(conn, "SELECT label_id FROM doc_labels WHERE doc_id = ?", &sqlitex.ExecOptions{
		Args: []interface{}{docID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			labelIDs = append(labelIDs, stmt.ColumnInt(0))
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch label IDs: %w", err)
	}

	for _, sentence := range sentences {
		err = sqlitex.Execute(conn, "INSERT INTO sentences (doc_id, sentence_id, data) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
			Args: []interface{}{docID, sentence.ID, string(sentence.Tokens)},
		})
		if err != nil {
			return fmt.Errorf("failed to insert sentence: %w", err)
		}
		sentRowID := conn.LastInsertRowID()

		for _, lemma := range sentence.Lemmas {
			err = sqlitex.Execute(conn, "INSERT INTO sentence_lemmas (lemma, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{lemma, sentRowID},
			})
			if err != nil {
				return fmt.Errorf("failed to insert lemma: %w", err)
			}
		}

		for _, labelID := range labelIDs {
			err = sqlitex.Execute(conn, "INSERT INTO sentence_labels (label_id, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{labelID, sentRowID},
			})
			if err != nil {
				return fmt.Errorf("failed to insert sentence label: %w", err)
			}
		}
	}

	return nil
}

func (h *DocStore) AddLabel(docID string, labels ...string) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	for _, label := range labels {
		labelID, err := h.upsertLabel(conn, label)
		if err != nil {
			return err
		}

		err = sqlitex.Execute(conn, "INSERT OR IGNORE INTO doc_labels (doc_id, label_id) VALUES (?, ?)", &sqlitex.ExecOptions{
			Args: []interface{}{docID, labelID},
		})
		if err != nil {
			return err
		}

		// Denormalize to sentence_labels
		err = sqlitex.Execute(conn, `
			INSERT OR IGNORE INTO sentence_labels (label_id, sentence_rowid)
			SELECT ?, rowid FROM sentences WHERE doc_id = ?`,
			&sqlitex.ExecOptions{
				Args: []interface{}{labelID, docID},
			})
		if err != nil {
			return err
		}
	}

	return nil
}

func (h *DocStore) RemoveLabel(docID string, labels ...string) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	for _, label := range labels {
		var labelID int
		err = sqlitex.Execute(conn, "SELECT id FROM labels WHERE name = ?", &sqlitex.ExecOptions{
			Args: []interface{}{label},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				labelID = stmt.ColumnInt(0)
				return nil
			},
		})
		if err != nil {
			return err
		}

		if labelID == 0 {
			continue // Label not found, nothing to remove
		}

		err = sqlitex.Execute(conn, "DELETE FROM doc_labels WHERE doc_id = ? AND label_id = ?", &sqlitex.ExecOptions{
			Args: []interface{}{docID, labelID},
		})
		if err != nil {
			return err
		}

		// Denormalize removal: remove from sentence_labels
		err = sqlitex.Execute(conn, `
			DELETE FROM sentence_labels 
			WHERE label_id = ? 
			AND sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)`,
			&sqlitex.ExecOptions{
				Args: []interface{}{labelID, docID},
			})
		if err != nil {
			return err
		}
	}

	return nil
}

func Labels(s string) []string {
	if s == "" {
		return nil
	}
	var res []string
	for _, part := range strings.Split(s, ",") {
		if part != "" {
			res = append(res, part)
		}
	}
	return res
}

// Exists returns true if a document with the given ID is present in the docs table.
func (h *DocStore) Exists(id string) (bool, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return false, err
	}
	defer h.pool.Put(conn)

	var exists bool
	err = sqlitex.Execute(conn,
		"SELECT 1 FROM docs WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				exists = true
				return nil
			},
		})
	return exists, err
}
