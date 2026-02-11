package zombiezen

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
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

func (h *DocStore) List(labelMatch string) ([]sent.Doc, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var docs []sent.Doc
	err = sqlitex.Execute(conn, "SELECT id, title, labels FROM docs ORDER BY title", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			labels := Labels(stmt.ColumnText(2))

			if labelMatch != "" {
				if !SliceElementsContains(labels, labelMatch) {
					return nil
				}
			}

			docs = append(docs, sent.Doc{
				Id:     stmt.ColumnInt(0),
				Title:  stmt.ColumnText(1),
				Labels: labels,
			})
			return nil
		},
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func SliceElementsContains(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func (h *DocStore) Read(id int) (sent.Doc, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return sent.Doc{}, err
	}
	defer h.pool.Put(conn)

	doc := sent.Doc{Id: id}
	found := false

	err = sqlitex.Execute(conn, "SELECT sentence_id, data FROM sentences WHERE doc_id = ? ORDER BY sentence_id", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			found = true
			sentenceID := stmt.ColumnInt(0)
			data := stmt.ColumnText(1)
			var tokens []sent.Token
			if err := json.Unmarshal([]byte(data), &tokens); err != nil {
				return err
			}
			doc.Sentences = append(doc.Sentences, sent.Sentence{
				Id:     sentenceID,
				DocId:  id,
				Tokens: tokens,
			})
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

func (h *DocStore) FindCandidates(lemmas []string, labels []string, after storage.Cursor, limit int, onCandidate func(sent.Sentence) error) (storage.Cursor, error) {
	if len(lemmas) == 0 {
		return after, nil
	}

	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return after, err
	}
	defer h.pool.Put(conn)

	query, args := h.buildCandidateQuery(lemmas, labels, after, limit)

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
				DocId: stmt.ColumnInt(1),
				Id:    stmt.ColumnInt(2),
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

func (h *DocStore) buildCandidateQuery(lemmas []string, labels []string, after storage.Cursor, limit int) (string, []interface{}) {
	var queryBuilder strings.Builder
	var args []interface{}

	// Handle Lemmas (mandatory)
	for i, lemma := range lemmas {
		if i > 0 {
			queryBuilder.WriteString(" INTERSECT ")
		}
		queryBuilder.WriteString("SELECT sentence_rowid FROM sentence_lemmas WHERE lemma = ? AND sentence_rowid > ?")
		args = append(args, lemma, after)
	}

	// Handle Labels (optional)
	for _, label := range labels {
		queryBuilder.WriteString(" INTERSECT ")
		queryBuilder.WriteString("SELECT sentence_rowid FROM sentence_labels WHERE label = ? AND sentence_rowid > ?")
		args = append(args, label, after)
	}

	queryBuilder.WriteString(" LIMIT ?")
	args = append(args, limit)

	return queryBuilder.String(), args
}

func (h *DocStore) Labels(pattern string) ([]string, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	labelMap := make(map[string]bool)
	err = sqlitex.Execute(conn, "SELECT labels FROM docs", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			for _, label := range Labels(stmt.ColumnText(0)) {
				if pattern != "" {
					if !strings.Contains(label, pattern) {
						continue
					}
				}
				labelMap[label] = true
			}
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(labelMap))
	for label := range labelMap {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	return labels, nil
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

	for _, sentence := range doc.Sentences {
		data, marshalErr := json.Marshal(sentence.Tokens)
		if marshalErr != nil {
			return marshalErr
		}

		err = sqlitex.Execute(conn, "INSERT INTO sentences (doc_id, sentence_id, data) VALUES (?, ?, ?)", &sqlitex.ExecOptions{
			Args: []interface{}{docID, sentence.Id, string(data)},
		})
		if err != nil {
			return fmt.Errorf("failed to insert sentence: %w", err)
		}
		sentRowID := conn.LastInsertRowID()

		// Extract unique lemmas
		uniqueLemmas := make(map[string]bool)
		for _, token := range sentence.Tokens {
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

		// Insert labels
		for _, label := range doc.Labels {
			err = sqlitex.Execute(conn, "INSERT INTO sentence_labels (label, sentence_rowid) VALUES (?, ?)", &sqlitex.ExecOptions{
				Args: []interface{}{label, sentRowID},
			})
			if err != nil {
				return fmt.Errorf("failed to insert label: %w", err)
			}
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
