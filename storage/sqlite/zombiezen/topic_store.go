package zombiezen

import (
	"context"
	"encoding/json"
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

func (h *TopicStore) ReadAll() (topic.Library, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var topics topic.Library
	query := fmt.Sprintf("SELECT name, exprs FROM %s WHERE user_id = ''", h.tableName)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(0)
			exprsJSON := stmt.ColumnText(1)

			var exprs []topic.TopicExpr
			if err := json.Unmarshal([]byte(exprsJSON), &exprs); err != nil {
				return err
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

func (h *TopicStore) Read(name string) (topic.Topic, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return topic.Topic{}, err
	}
	defer h.pool.Put(conn)

	var t topic.Topic
	found := false
	query := fmt.Sprintf("SELECT name, exprs FROM %s WHERE user_id = '' AND name = ? LIMIT 1", h.tableName)
	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{name},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(0)
			exprsJSON := stmt.ColumnText(1)

			var exprs []topic.TopicExpr
			if err := json.Unmarshal([]byte(exprsJSON), &exprs); err != nil {
				return err
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

func (h *TopicStore) Write(tp topic.Topic) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	exprsJSON, err := json.Marshal(tp.Exprs)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (user_id, name, exprs, updated)
		VALUES ('', ?, ?, strftime('%s', 'now'))
		ON CONFLICT(user_id, name) DO UPDATE SET
			exprs = excluded.exprs,
			updated = excluded.updated
	`, h.tableName, "%Y-%m-%dT%H:%M:%SZ")

	err = sqlitex.Execute(conn, query, &sqlitex.ExecOptions{
		Args: []interface{}{tp.Name, string(exprsJSON)},
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

// assembleTopic sets TopicName, ExprIndex and ItemIndex for each item
func (h *TopicStore) assembleTopic(name string, exprs []topic.TopicExpr) topic.Topic {
	for index := range exprs {
		for idx := range exprs[index] {
			exprs[index][idx].TopicName = name
			exprs[index][idx].ExprIndex = index
			exprs[index][idx].ItemIndex = idx
		}
	}
	return topic.Topic{
		Name:  name,
		Exprs: exprs,
	}
}
