package zombiezen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	sent "github.com/revelaction/segrob/sentence"
	"github.com/revelaction/segrob/storage"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type DocHandler struct {
	pool *sqlitex.Pool
}

var _ storage.DocRepository = (*DocHandler)(nil)

func NewDocHandler(pool *sqlitex.Pool) *DocHandler {
	return &DocHandler{pool: pool}
}

func (h *DocHandler) List() ([]sent.Doc, error) {
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

func (h *DocHandler) Doc(id int) (sent.Doc, error) {
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

func (h *DocHandler) FindCandidates(lemmas []string, after storage.Cursor, limit int) ([]storage.SentenceResult, storage.Cursor, error) {
	if len(lemmas) == 0 {
		return nil, after, nil
	}

	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, after, err
	}
	defer h.pool.Put(conn)

	// Build query dynamically based on number of lemmas
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

	results := make([]storage.SentenceResult, 0, len(rowIDs))
	newCursor := after

	for _, rowID := range rowIDs {
		// Update cursor to the last processed rowID
		if storage.Cursor(rowID) > newCursor {
			newCursor = storage.Cursor(rowID)
		}

		var res storage.SentenceResult
		res.RowID = rowID

		err = sqlitex.Execute(conn, "SELECT s.doc_id, s.data, d.title FROM sentences s JOIN docs d ON s.doc_id = d.id WHERE s.rowid = ?", &sqlitex.ExecOptions{
			Args: []interface{}{rowID},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				res.DocID = stmt.ColumnInt(0)
				data := stmt.ColumnText(1)
				res.DocTitle = stmt.ColumnText(2)

				if err := json.Unmarshal([]byte(data), &res.Tokens); err != nil {
					return err
				}
				return nil
			},
		})
		if err != nil {
			return nil, after, err
		}
		results = append(results, res)
	}

	return results, newCursor, nil
}

func (h *DocHandler) WriteDoc(doc sent.Doc) error {
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
