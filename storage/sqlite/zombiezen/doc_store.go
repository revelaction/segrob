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

func (h *DocStore) List() ([]sent.Doc, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var docs []sent.Doc
	err = sqlitex.Execute(conn, "SELECT id, title, labels FROM docs ORDER BY title", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			doc := sent.Doc{
				Id:    stmt.ColumnInt(0),
				Title: stmt.ColumnText(1),
			}
			labelsStr := stmt.ColumnText(2)
			if labelsStr != "" {
				doc.Labels = strings.Split(labelsStr, ",")
			}
			docs = append(docs, doc)
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (h *DocStore) Read(id int) (sent.Doc, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return sent.Doc{}, err
	}
	defer h.pool.Put(conn)

	doc := sent.Doc{Id: id}
	found := false

	err = sqlitex.Execute(conn, "SELECT data FROM sentences WHERE doc_id = ? ORDER BY rowid", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			data := stmt.ColumnText(0)
			var tokens []sent.Token
			if err := json.Unmarshal([]byte(data), &tokens); err != nil {
				return err
			}
			doc.Tokens = append(doc.Tokens, tokens)
			return nil
		},
	})
	if err != nil {
		return sent.Doc{}, err
	}
	if !found {
		return sent.Doc{}, fmt.Errorf("doc not found: %d", id)
	}

	return doc, nil
}

func (h *DocStore) FindCandidates(lemmas []string, after storage.Cursor, limit int) ([]storage.SentenceResult, storage.Cursor, error) {
	if len(lemmas) == 0 {
		return nil, after, nil
	}

	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, after, err
	}
	defer h.pool.Put(conn)

	// Build query dynamically based on number of lemmas.
	// We use INTERSECT to ensure that we only get sentence_rowids that contain ALL lemmas.
	// Note: INTERSECT also guarantees that the resulting set of rowIDs is unique.
	var queryBuilder strings.Builder
	var args []interface{}

	for i, lemma := range lemmas {
		if i > 0 {
			queryBuilder.WriteString(" INTERSECT ")
		}
		queryBuilder.WriteString("SELECT sentence_rowid FROM sentence_lemmas WHERE lemma = ? AND sentence_rowid > ?")
		args = append(args, lemma, after)
	}
	queryBuilder.WriteString(" LIMIT ?")
	args = append(args, limit)

	// We need to fetch the rowIDs first
	var rowIDs []int64
	err = sqlitex.Execute(conn, queryBuilder.String(), &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			rowIDs = append(rowIDs, stmt.ColumnInt64(0))
			return nil
		},
	})
	if err != nil {
		return nil, after, err
	}

	if len(rowIDs) == 0 {
		return nil, after, nil
	}

	// TODO: Consolidate into a single query using a subquery for better performance.
	// For now, we use a second bulk query to fetch the sentence data.
	idStrings := make([]string, len(rowIDs))
	for i, id := range rowIDs {
		idStrings[i] = strconv.FormatInt(id, 10)
	}
	idList := strings.Join(idStrings, ",")

	results := make([]storage.SentenceResult, 0, len(rowIDs))
	query := fmt.Sprintf("SELECT rowid, doc_id, data FROM sentences WHERE rowid IN (%s) ORDER BY rowid", idList)

	newCursor := after
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			rowID := stmt.ColumnInt64(0)
			if storage.Cursor(rowID) > newCursor {
				newCursor = storage.Cursor(rowID)
			}

			res := storage.SentenceResult{
				RowID: rowID,
				DocID: stmt.ColumnInt(1),
			}
			data := stmt.ColumnText(2)
			if err := json.Unmarshal([]byte(data), &res.Tokens); err != nil {
				return err
			}
			results = append(results, res)
			return nil
		},
	})
	if err != nil {
		return nil, after, err
	}

	return results, newCursor, nil
}

func (h *DocStore) Write(doc sent.Doc) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	// Insert Doc
	labels := strings.Join(doc.Labels, ",")
	err = sqlitex.Execute(conn, "INSERT INTO docs (title, labels) VALUES (?, ?)", &sqlitex.ExecOptions{
		Args: []interface{}{doc.Title, labels},
	})
	if err != nil {
		return fmt.Errorf("failed to insert doc: %w", err)
	}
	docID := conn.LastInsertRowID()

	for _, sentence := range doc.Tokens {
		data, marshalErr := json.Marshal(sentence)
		if marshalErr != nil {
			return marshalErr
		}

		err = sqlitex.Execute(conn, "INSERT INTO sentences (doc_id, data) VALUES (?, ?)", &sqlitex.ExecOptions{
			Args: []interface{}{docID, string(data)},
		})
		if err != nil {
			return fmt.Errorf("failed to insert sentence: %w", err)
		}
		sentRowID := conn.LastInsertRowID()

		// Extract unique lemmas
		uniqueLemmas := make(map[string]bool)
		for _, token := range sentence {
			if token.Lemma != "" {
				uniqueLemmas[token.Lemma] = true
			}
		}

		for lemma := range uniqueLemmas {
			err = sqlitex.Execute(conn, "INSERT INTO sentence_lemmas (lemma, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{lemma, sentRowID},
			})
			if err != nil {
				return fmt.Errorf("failed to insert lemma: %w", err)
			}
		}
	}

	return nil
}
