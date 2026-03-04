package zombiezen

import (
	"context"
	"fmt"

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

func (s *CorpusStore) List() ([]storage.CorpusRecord, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer s.pool.Put(conn)

	var records []storage.CorpusRecord
	err = sqlitex.Execute(conn,
		"SELECT id, labels FROM corpus",
		&sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				records = append(records, storage.CorpusRecord{
					ID:     stmt.ColumnText(0),
					Labels: stmt.ColumnText(1),
				})
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
			`INSERT INTO corpus (id, labels, epub, txt, txt_hash, txt_reviewed, txt_reviewed_at, txt_reviewer, txt_review_notes, deleted, deleted_at)
			 VALUES (?, ?, ?, ?, ?, false, '', '', '', false, '')`,
			&sqlitex.ExecOptions{
				Args: []interface{}{record.ID, record.Labels, record.Epub, record.Txt, record.TxtHash},
			})
		if err != nil {
			return fmt.Errorf("failed to insert corpus record %s: %w", record.ID, err)
		}
	}

	return nil
}

// ReadMeta retrieves id, epub, and labels for a given document ID.
func (s *CorpusStore) ReadMeta(id string) (storage.CorpusMeta, error) {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return storage.CorpusMeta{}, err
	}
	defer s.pool.Put(conn)

	var meta storage.CorpusMeta
	var found bool
	err = sqlitex.Execute(conn,
		"SELECT id, epub, labels FROM corpus WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				meta.ID = stmt.ColumnText(0)
				meta.Epub = stmt.ColumnText(1)
				meta.Labels = stmt.ColumnText(2)
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
		"UPDATE corpus SET nlp = ? WHERE id = ?",
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

func (s *CorpusStore) ClearNlp(id string) error {
	conn, err := s.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer s.pool.Put(conn)

	return sqlitex.Execute(conn,
		"UPDATE corpus SET nlp = NULL WHERE id = ?",
		&sqlitex.ExecOptions{
			Args: []interface{}{id},
		})
}
