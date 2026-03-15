package zombiezen

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/revelaction/segrob/storage"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

var _ storage.CorpusRepository = (*CorpusStore)(nil)

type CorpusStore struct {
	pool *sqlitex.Pool
}

func NewCorpusStore(pool *sqlitex.Pool) *CorpusStore {
	return &CorpusStore{pool: pool}
}

// Exists returns true if a record with the given ID is present in the docs table.
func (s *CorpusStore) Exists(id string) (bool, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return false, err
	}
	defer s.pool.Put(conn)

	var exists bool
	err = sqlitex.Execute(conn,
		"SELECT 1 FROM corpus WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				exists = true
				return nil
			},
		})
	return exists, err
}

func (s *CorpusStore) List() ([]storage.CorpusMeta, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var records []storage.CorpusMeta
	err = sqlitex.Execute(conn,
		`SELECT 
			id, labels, epub, txt_hash, txt_created_at, 
			txt_edit, txt_edit_at, txt_edit_by, txt_edit_notes, 
			txt_ack, txt_ack_at, txt_ack_by,
			nlp_created_at, nlp_ack, nlp_ack_at, nlp_ack_by, 
			deleted_at, created_at, updated_at 
		 FROM corpus`,
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				meta, err := scanCorpusMeta(stmt)
				if err != nil {
					return err
				}
				records = append(records, meta)
				return nil
			},
		})
	if err != nil {
		return nil, err
	}
	return records, nil
}

// WriteStream inserts corpus records yielded by the iterator into
// a single transaction. If the iterator yields an error or a DB insert
// fails, the transaction is rolled back.
func (s *CorpusStore) WriteStream(seq func(yield func(storage.CorpusRecord, error) bool)) (err error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	// Start Transaction
	defer sqlitex.Save(conn)(&err)

	for record, iterErr := range seq {
		if iterErr != nil {
			return iterErr
		}

		err = sqlitex.Execute(conn,
			`INSERT INTO corpus (id, labels, epub, txt, txt_hash)
			 VALUES (?, ?, ?, ?, ?)`,
			&sqlitex.ExecOptions{
				Args: []interface{}{record.ID, record.Labels, record.Epub, record.Txt, record.TxtHash},
			})
		if err != nil {
			return fmt.Errorf("failed to insert corpus record %s: %w", record.ID, err)
		}
	}

	return nil
}

// ReadMeta retrieves full metadata for a given document ID.
func (s *CorpusStore) ReadMeta(id string) (storage.CorpusMeta, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return storage.CorpusMeta{}, err
	}
	defer s.pool.Put(conn)

	var meta storage.CorpusMeta
	var found bool
	err = sqlitex.Execute(conn,
		`SELECT 
			id, labels, epub, txt_hash, txt_created_at, 
			txt_edit, txt_edit_at, txt_edit_by, txt_edit_notes, 
			txt_ack, txt_ack_at, txt_ack_by,
			nlp_created_at, nlp_ack, nlp_ack_at, nlp_ack_by, 
			deleted_at, created_at, updated_at 
		 FROM corpus WHERE id = ?`,
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				var err error
				meta, err = scanCorpusMeta(stmt)
				if err != nil {
					return err
				}
				found = true
				return nil
			},
		})
	if err != nil {
		return storage.CorpusMeta{}, err
	}
	if !found {
		return storage.CorpusMeta{}, fmt.Errorf("document %s not found in corpus", id)
	}
	return meta, nil
}

// scanCorpusMeta helper to scan a row into CorpusMeta
func scanCorpusMeta(stmt *sqlite.Stmt) (storage.CorpusMeta, error) {
	txtCreatedAt, err := storage.TimeParse(stmt.GetText("txt_created_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing txt_created_at: %w", err)
	}
	txtEditAt, err := storage.TimeParse(stmt.GetText("txt_edit_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing txt_edit_at: %w", err)
	}
	txtAckAt, err := storage.TimeParse(stmt.GetText("txt_ack_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing txt_ack_at: %w", err)
	}
	nlpCreatedAt, err := storage.TimeParse(stmt.GetText("nlp_created_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing nlp_created_at: %w", err)
	}
	nlpAckAt, err := storage.TimeParse(stmt.GetText("nlp_ack_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing nlp_ack_at: %w", err)
	}
	deletedAt, err := storage.TimeParse(stmt.GetText("deleted_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing deleted_at: %w", err)
	}
	createdAt, err := storage.TimeParse(stmt.GetText("created_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing created_at: %w", err)
	}
	updatedAt, err := storage.TimeParse(stmt.GetText("updated_at"))
	if err != nil {
		return storage.CorpusMeta{}, fmt.Errorf("error parsing updated_at: %w", err)
	}

	return storage.CorpusMeta{
		ID:           stmt.GetText("id"),
		Labels:       stmt.GetText("labels"),
		Epub:         stmt.GetText("epub"),
		TxtHash:      stmt.GetText("txt_hash"),
		TxtCreatedAt: txtCreatedAt,
		TxtEdit:      stmt.GetBool("txt_edit"),
		TxtEditAt:    txtEditAt,
		TxtEditBy:    stmt.GetText("txt_edit_by"),
		TxtEditNotes: stmt.GetText("txt_edit_notes"),
		TxtAck:       stmt.GetBool("txt_ack"),
		TxtAckAt:     txtAckAt,
		TxtAckBy:     stmt.GetText("txt_ack_by"),
		NlpCreatedAt: nlpCreatedAt,
		NlpAck:       stmt.GetBool("nlp_ack"),
		NlpAckAt:     nlpAckAt,
		NlpAckBy:     stmt.GetText("nlp_ack_by"),
		DeletedAt:    deletedAt,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

// ReadTxt retrieves the txt field for a given document ID as raw bytes.
func (s *CorpusStore) ReadTxt(id string) ([]byte, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var txt []byte
	var found bool
	err = sqlitex.Execute(conn,
		"SELECT txt FROM corpus WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				// ColumnBytes copies the blob/text content as raw bytes
				n := stmt.ColumnLen(0)
				txt = make([]byte, n)
				stmt.ColumnBytes(0, txt)
				found = true
				return nil
			},
		})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("document %s not found in corpus", id)
	}
	return txt, nil
}

func (s *CorpusStore) WriteNlp(id string, nlp []byte) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		"UPDATE corpus SET nlp = ?, nlp_created_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{string(nlp), id},
		})
}

func (s *CorpusStore) ReadNlp(id string) ([]byte, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var nlp []byte
	var found bool
	err = sqlitex.Execute(conn,
		"SELECT nlp FROM corpus WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				n := stmt.ColumnLen(0)
				if n > 0 {
					nlp = make([]byte, n)
					stmt.ColumnBytes(0, nlp)
				}
				found = true
				return nil
			},
		})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("document %s not found in corpus", id)
	}
	return nlp, nil
}

// ListLabels returns all labels (unique names) found in the corpus.
func (s *CorpusStore) ListLabels(labelSubStr string) ([]string, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	lblMap := make(map[string]bool)

	err = sqlitex.Execute(conn, "SELECT labels FROM corpus WHERE labels != ''", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			labelsStr := stmt.ColumnText(0)
			for _, lbl := range strings.Split(labelsStr, ",") {
				if strings.Contains(lbl, labelSubStr) {
					lblMap[lbl] = true
				}
			}
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list corpus labels: %w", err)
	}

	keys := slices.Sorted(maps.Keys(lblMap))
	return keys, nil
}

func (s *CorpusStore) ClearNlp(id string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		"UPDATE corpus SET nlp = '', updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
		})
}

func (s *CorpusStore) UpdateTxt(id string, txt []byte, txtHash string, by string, notes string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		`UPDATE corpus SET 
			txt = ?, 
			txt_hash = ?, 
			txt_edit = 1, 
			txt_edit_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), 
			txt_edit_by = ?, 
			txt_edit_notes = ?, 
			updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) 
		 WHERE id = ?`,
		&sqlitex.ExecOptions{
			Args: []interface{}{string(txt), txtHash, by, notes, id},
		})
}

func (s *CorpusStore) AckTxt(id string, by string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		`UPDATE corpus SET 
			txt_ack = 1, 
			txt_ack_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), 
			txt_ack_by = ?, 
			updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) 
		 WHERE id = ?`,
		&sqlitex.ExecOptions{
			Args: []interface{}{by, id},
		})
}

func (s *CorpusStore) AckNlp(id string, by string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		`UPDATE corpus SET 
			nlp_ack = 1, 
			nlp_ack_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), 
			nlp_ack_by = ?, 
			updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) 
		 WHERE id = ?`,
		&sqlitex.ExecOptions{
			Args: []interface{}{by, id},
		})
}

// AddLabel adds labels to a document in the corpus.
func (s *CorpusStore) AddLabel(id string, labels ...string) (err error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	var currentLabelsStr string
	var found bool
	err = sqlitex.Execute(conn, "SELECT labels FROM corpus WHERE id = ?", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			currentLabelsStr = stmt.ColumnText(0)
			found = true
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to read corpus %s: %w", id, err)
	}
	if !found {
		return fmt.Errorf("document %s not found in corpus", id)
	}

	lblMap := make(map[string]bool)
	if currentLabelsStr != "" {
		for _, lbl := range strings.Split(currentLabelsStr, ",") {
			lblMap[lbl] = true
		}
	}

	for _, lbl := range labels {
		lblMap[storage.NormalizeLabel(lbl)] = true
	}

	keys := slices.Sorted(maps.Keys(lblMap))

	newLabelsStr := strings.Join(keys, ",")
	if newLabelsStr != currentLabelsStr {
		err = sqlitex.Execute(conn,
			"UPDATE corpus SET labels = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) WHERE id = ?",
			&sqlitex.ExecOptions{
				Args: []interface{}{newLabelsStr, id},
			})
		if err != nil {
			return fmt.Errorf("failed to update labels for corpus %s: %w", id, err)
		}
	}

	return nil
}

// DeleteLabel deletes labels from a document in the corpus.
func (s *CorpusStore) DeleteLabel(id string, labels ...string) (err error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	var currentLabelsStr string
	var found bool
	err = sqlitex.Execute(conn, "SELECT labels FROM corpus WHERE id = ?", &sqlitex.ExecOptions{
		Args: []interface{}{id},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			currentLabelsStr = stmt.ColumnText(0)
			found = true
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to read corpus %s: %w", id, err)
	}
	if !found {
		return fmt.Errorf("document %s not found in corpus", id)
	}

	if currentLabelsStr == "" {
		return nil
	}

	lblMap := make(map[string]bool)
	for _, lbl := range strings.Split(currentLabelsStr, ",") {
		lblMap[lbl] = true
	}

	for _, lbl := range labels {
		delete(lblMap, lbl)
	}

	keys := slices.Sorted(maps.Keys(lblMap))

	newLabelsStr := strings.Join(keys, ",")
	if newLabelsStr != currentLabelsStr {
		err = sqlitex.Execute(conn,
			"UPDATE corpus SET labels = ?, updated_at = (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')) WHERE id = ?",
			&sqlitex.ExecOptions{
				Args: []interface{}{newLabelsStr, id},
			})
		if err != nil {
			return fmt.Errorf("failed to update labels for corpus %s: %w", id, err)
		}
	}

	return nil
}

// Delete removes a document from the corpus by its ID.
func (s *CorpusStore) Delete(id string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		"DELETE FROM corpus WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
		})
}
