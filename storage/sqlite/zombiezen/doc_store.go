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

func (h *DocStore) insertSentence(conn *sqlite.Conn, docID string, s storage.SentenceIngest) (int64, error) {
	err := sqlitex.Execute(conn, "INSERT INTO sentences (doc_id, sentence_id, data) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{docID, s.ID, string(s.Tokens)},
	})
	if err != nil {
		return 0, err
	}
	return conn.LastInsertRowID(), nil
}

func (h *DocStore) insertLemmaOptimize(conn *sqlite.Conn, sentenceRowID int64, lemma string) error {
	return sqlitex.Execute(conn, "INSERT INTO sentence_lemmas (lemma, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{lemma, sentenceRowID},
	})
}

func (h *DocStore) insertLabelOptimize(conn *sqlite.Conn, sentenceRowID int64, labelID int) error {
	return sqlitex.Execute(conn, "INSERT INTO sentence_labels (label_id, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{labelID, sentenceRowID},
	})
}

func (h *DocStore) List() ([]sent.Meta, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var metas []sent.Meta

	err = sqlitex.Execute(conn, "SELECT id, source, label_ids FROM docs ORDER BY source", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			meta := sent.Meta{
				Id:     stmt.ColumnText(0),
				Source: stmt.ColumnText(1),
			}
			
			labelIDsStr := stmt.ColumnText(2)
			var labelIDs []int
			if labelIDsStr != "" {
				for _, idStr := range strings.Split(labelIDsStr, ",") {
					if id, err := strconv.Atoi(idStr); err == nil {
						labelIDs = append(labelIDs, id)
					}
				}
			}
			meta.LabelIDs = labelIDs
			
			metas = append(metas, meta)
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list docs: %w", err)
	}

	return metas, nil
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

// buildCandidateQuery constructs the SQL query for finding sentences matching multiple lemmas and labels.
//
// Example for lemmas ["house", "big"] and labels [1, 5]:
//
//	SELECT sentence_rowid FROM sentence_lemmas AS s_outer
//	WHERE lemma = ? AND sentence_rowid > ?
//	AND EXISTS (SELECT 1 FROM sentence_lemmas WHERE sentence_rowid = s_outer.sentence_rowid AND lemma = ?)
//	AND EXISTS (SELECT 1 FROM sentence_labels WHERE sentence_rowid = s_outer.sentence_rowid AND label_id = ?)
//	AND EXISTS (SELECT 1 FROM sentence_labels WHERE sentence_rowid = s_outer.sentence_rowid AND label_id = ?)
//	ORDER BY sentence_rowid ASC LIMIT ?
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

func (h *DocStore) ListLabels(labelSubStr string) (sent.Labels, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	query, args := h.buildListLabelsQuery(labelSubStr)

	labels := make(sent.Labels)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			// stmt.ColumnInt(0) is ID, stmt.ColumnText(1) is Name
			labels[stmt.ColumnText(1)] = stmt.ColumnInt(0)
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

func (h *DocStore) WriteMeta(id string, source string, labels []string) (labelIDs []int, err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	var labelIDStrs []string
	for _, label := range labels {
		labelID, err := h.upsertLabel(conn, label)
		if err != nil {
			return nil, fmt.Errorf("failed to upsert label %s: %w", label, err)
		}
		labelIDs = append(labelIDs, labelID)
		labelIDStrs = append(labelIDStrs, strconv.Itoa(labelID))
	}

	labelIDsStr := strings.Join(labelIDStrs, ",")

	err = sqlitex.Execute(conn, "INSERT INTO docs (id, source, label_ids) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{id, source, labelIDsStr},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert doc %s: %w", id, err)
	}

	return labelIDs, nil
}

func (h *DocStore) WriteNlpData(docID string, sentences []storage.SentenceIngest) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	for _, sentence := range sentences {
		_, err := h.insertSentence(conn, docID, sentence)
		if err != nil {
			return fmt.Errorf("failed to insert sentence: %w", err)
		}
	}
	return nil
}

// Signature change: receives labelIDs directly from WriteMeta
func (h *DocStore) WriteLabelsOptimization(docID string, labelIDs []int) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	for _, labelID := range labelIDs {
		err = sqlitex.Execute(conn,
			`INSERT INTO sentence_labels (label_id, sentence_rowid)
             SELECT ?, rowid FROM sentences WHERE doc_id = ?`,
			&sqlitex.ExecOptions{
				Args: []interface{}{labelID, docID},
			})
		if err != nil {
			return fmt.Errorf("failed to insert sentence labels: %w", err)
		}
	}

	return nil
}

func (h *DocStore) WriteLemmaOptimization(docID string, sentences []storage.SentenceIngest) (err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	// Build a map: sentence_id -> lemmas from the ingest data
	lemmaMap := make(map[int][]string)
	for _, s := range sentences {
		lemmaMap[s.ID] = s.Lemmas
	}

	// Fetch sentence rowids and sentence_ids
	err = sqlitex.Execute(conn, "SELECT rowid, sentence_id FROM sentences WHERE doc_id = ?", &sqlitex.ExecOptions{
		Args: []interface{}{docID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			rowID := stmt.ColumnInt64(0)
			sentID := stmt.ColumnInt(1)
			lemmas, ok := lemmaMap[sentID]
			if !ok {
				return nil
			}
			for _, lemma := range lemmas {
				if err := h.insertLemmaOptimize(conn, rowID, lemma); err != nil {
					return fmt.Errorf("failed to insert lemma: %w", err)
				}
			}
			return nil
		},
	})

	return err
}

func (h *DocStore) HasLabelsOptimization(id string) (bool, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return false, err
	}
	defer h.pool.Put(conn)

	var has bool
	err = sqlitex.Execute(conn,
		`SELECT 1 FROM sentence_labels
         WHERE sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)
         LIMIT 1`,
		&sqlitex.ExecOptions{
			Args:       []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error { has = true; return nil },
		})
	return has, err
}

func (h *DocStore) HasLemmaOptimization(id string) (bool, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return false, err
	}
	defer h.pool.Put(conn)

	var has bool
	err = sqlitex.Execute(conn,
		`SELECT 1 FROM sentence_lemmas
         WHERE sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)
         LIMIT 1`,
		&sqlitex.ExecOptions{
			Args:       []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error { has = true; return nil },
		})
	return has, err
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

// DeleteLemmaOptimization removes sentence_lemmas rows for docID.
func (h *DocStore) DeleteLemmaOptimization(docID string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	err = sqlitex.Execute(conn,
		`DELETE FROM sentence_lemmas WHERE sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)`,
		&sqlitex.ExecOptions{Args: []interface{}{docID}})
	if err != nil {
		return fmt.Errorf("failed to delete lemma optimization: %w", err)
	}
	return nil
}

// DeleteLabelsOptimization removes sentence_labels rows for docID.
func (h *DocStore) DeleteLabelsOptimization(docID string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	err = sqlitex.Execute(conn,
		`DELETE FROM sentence_labels WHERE sentence_rowid IN (SELECT rowid FROM sentences WHERE doc_id = ?)`,
		&sqlitex.ExecOptions{Args: []interface{}{docID}})
	if err != nil {
		return fmt.Errorf("failed to delete labels optimization: %w", err)
	}
	return nil
}

// DeleteNlpData removes all sentences rows for docID.
func (h *DocStore) DeleteNlpData(docID string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	err = sqlitex.Execute(conn,
		`DELETE FROM sentences WHERE doc_id = ?`,
		&sqlitex.ExecOptions{Args: []interface{}{docID}})
	if err != nil {
		return fmt.Errorf("failed to delete nlp data: %w", err)
	}
	return nil
}

// DeleteMeta removes the docs row for docID.
// Rows in the labels table are shared across documents and are not touched.
func (h *DocStore) DeleteMeta(docID string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	err = sqlitex.Execute(conn,
		`DELETE FROM docs WHERE id = ?`,
		&sqlitex.ExecOptions{Args: []interface{}{docID}})
	if err != nil {
		return fmt.Errorf("failed to delete meta: %w", err)
	}
	return nil
}
