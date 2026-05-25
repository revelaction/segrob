package zombiezen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type TopicStore struct {
	pool      *sqlitex.Pool
	tableName string
}

var _ storage.TopicReader = (*TopicStore)(nil)
var _ storage.TopicWriter = (*TopicStore)(nil)
var _ storage.TopicDeleter = (*TopicStore)(nil)

func NewLiveTopicStore(pool *sqlitex.Pool) *TopicStore {
	return &TopicStore{pool: pool, tableName: "topics"}
}

func NewCorpusTopicStore(pool *sqlitex.Pool) *TopicStore {
	return &TopicStore{pool: pool, tableName: "corpus_topics"}
}

func (h *TopicStore) ReadAll(userID string) (topic.Library, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var topics topic.Library
	query := fmt.Sprintf("SELECT name, exprs FROM %s WHERE user_id = ?", h.tableName)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{userID},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(0)
			exprsJSON := stmt.ColumnText(1)

			var exprs []topic.TopicExpr
			unmarshalErr := json.Unmarshal([]byte(exprsJSON), &exprs)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			topics = append(topics, h.assembleTopic(name, exprs))
			return nil
		},
	})

	if err != nil {
		return nil, err
	}

	return topics, nil
}

func (h *TopicStore) Read(userID string, name string) (topic.Topic, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return topic.Topic{}, err
	}
	defer h.pool.Put(conn)

	var t topic.Topic
	found := false
	query := fmt.Sprintf("SELECT name, exprs FROM %s WHERE user_id = ? AND name = ? LIMIT 1", h.tableName)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{userID, name},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(0)
			exprsJSON := stmt.ColumnText(1)

			var exprs []topic.TopicExpr
			unmarshalErr := json.Unmarshal([]byte(exprsJSON), &exprs)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			t = h.assembleTopic(name, exprs)
			found = true
			return nil
		},
	})

	if err != nil {
		return topic.Topic{}, err
	}

	if !found {
		return topic.Topic{}, fmt.Errorf("topic not found: %s", name)
	}

	return t, nil
}

// Upsert provides both an atomic read-modify-write operation and a direct write operation.
// It can be used both to create a new topic and to update an existing one.
//
// If the topic does not exist for the given userID and name, it initializes a new
// topic.Topic with that name and empty expressions. The provided mutation function 'fn'
// is then called with this (possibly empty) topic. If 'fn' returns storage.ErrNoChange,
// the operation is aborted and the current state is returned.
//
// Example usage for adding an expression (Insert or Update):
//
//	updated, err := store.Upsert(userID, topic.Topic{Name: "my-topic"}, func(t topic.Topic) (topic.Topic, error) {
//	    for _, existing := range t.Exprs {
//	        if topic.EqualExpr(existing, newExpr) {
//	            return t, storage.ErrNoChange
//	        }
//	    }
//	    t.Exprs = append(t.Exprs, newExpr)
//	    return t, nil
//	})
//
// If 'fn' is nil, it performs a direct write/overwrite using 'tp', skipping the SELECT.
// If 'fn' is provided, it fetches the existing topic for the name in 'tp.Name' (or
// initializes an empty one), passes it to 'fn' for mutation, and saves the result.
func (h *TopicStore) Upsert(userID string, tp topic.Topic, fn func(topic.Topic) (topic.Topic, error)) (result topic.Topic, err error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return topic.Topic{}, err
	}
	defer h.pool.Put(conn)

	defer sqlitex.Save(conn)(&err)

	var target topic.Topic

	// Branch on the existence of the mutation function
	if fn == nil {
		// Pure Write: Skip SELECT. Just use the provided topic.
		target = tp
	} else {
		// Mutate: Perform SELECT and apply the callback.
		current := topic.Topic{Name: tp.Name}
		readQuery := fmt.Sprintf(`SELECT exprs FROM %s WHERE user_id = ? AND name = ?`, h.tableName)
		err = sqlitex.Execute(conn, readQuery, &sqlitex.ExecOptions{
			Args: []interface{}{userID, tp.Name},
			ResultFunc: func(stmt *sqlite.Stmt) error {
				return json.Unmarshal([]byte(stmt.ColumnText(0)), &current.Exprs)
			},
		})
		if err != nil {
			return topic.Topic{}, err
		}

		target, err = fn(current)
		if errors.Is(err, storage.ErrNoChange) {
			return current, nil
		}
		if err != nil {
			return topic.Topic{}, err
		}
	}

	// Single, unified write execution
	exprsJSON, err := json.Marshal(target.Exprs)
	if err != nil {
		return topic.Topic{}, err
	}

	upsertQuery := fmt.Sprintf(`
		INSERT INTO %s (user_id, name, exprs, updated)
		VALUES (?, ?, ?, strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ', 'now'))
		ON CONFLICT(user_id, name) DO UPDATE SET
			exprs   = excluded.exprs,
			updated = excluded.updated
		RETURNING name, exprs
	`, h.tableName)

	err = sqlitex.Execute(conn, upsertQuery, &sqlitex.ExecOptions{
		Args: []interface{}{userID, target.Name, string(exprsJSON)},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			returnedName := stmt.ColumnText(0)
			
			var returnedExprs []topic.TopicExpr
			unmarshalErr := json.Unmarshal([]byte(stmt.ColumnText(1)), &returnedExprs)
			if unmarshalErr != nil {
				return unmarshalErr
			}
			result = h.assembleTopic(returnedName, returnedExprs)
			return nil
		},
	})
	if err != nil {
		return topic.Topic{}, err
	}

	return result, nil
}

func (h *TopicStore) CopyDefault(userID string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, name, exprs)
		SELECT ?, name, exprs FROM %s WHERE user_id = ''
		ON CONFLICT(user_id, name) DO NOTHING
	`, h.tableName, h.tableName)

	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{userID},
	})

	return err
}

func (h *TopicStore) Delete(name string) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	query := fmt.Sprintf("DELETE FROM %s WHERE user_id = '' AND name = ?", h.tableName)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{name},
	})

	return err
}

func (h *TopicStore) assembleTopic(name string, exprs []topic.TopicExpr) topic.Topic {
	return topic.Topic{
		Name:  name,
		Exprs: exprs,
	}
}
