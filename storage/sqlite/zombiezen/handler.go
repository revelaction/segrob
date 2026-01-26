package zombiezen

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/revelaction/segrob/storage"
	"github.com/revelaction/segrob/topic"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

type TopicHandler struct {
	pool *sqlitex.Pool
}

var _ storage.TopicReader = (*TopicHandler)(nil)
var _ storage.TopicWriter = (*TopicHandler)(nil)

func NewTopicHandler(dbPath string) (*TopicHandler, error) {
	// Follow NewZombiezenPool conventions from dev/sqlite_zombiezen.go
	poolSize := runtime.NumCPU()
	initString := fmt.Sprintf("file:%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", dbPath)

	pool, err := sqlitex.NewPool(initString, sqlitex.PoolOptions{
		PoolSize: poolSize,
		PrepareConn: func(conn *sqlite.Conn) error {
			return InitSchema(conn)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create zombiezen pool at %s: %w", dbPath, err)
	}

	return &TopicHandler{pool: pool}, nil
}

func (h *TopicHandler) Close() error {
	return h.pool.Close()
}

func (h *TopicHandler) All() ([]topic.Topic, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var topics []topic.Topic
	err = sqlitex.Execute(conn, "SELECT name, exprs FROM topics WHERE user_id IS NULL", &sqlitex.ExecOptions{
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

func (h *TopicHandler) Names() ([]string, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return nil, err
	}
	defer h.pool.Put(conn)

	var names []string
	err = sqlitex.Execute(conn, "SELECT name FROM topics WHERE user_id IS NULL ORDER BY name", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			names = append(names, stmt.ColumnText(0))
			return nil
		},
	})

	if err != nil {
		return nil, err
	}

	return names, nil
}

func (h *TopicHandler) Topic(name string) (topic.Topic, error) {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return topic.Topic{}, err
	}
	defer h.pool.Put(conn)

	var t topic.Topic
	found := false
	err = sqlitex.Execute(conn, "SELECT name, exprs FROM topics WHERE user_id IS NULL AND name = ? LIMIT 1", &sqlitex.ExecOptions{
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

func (h *TopicHandler) Write(tp topic.Topic) error {
	conn, err := h.pool.Take(context.TODO())
	if err != nil {
		return err
	}
	defer h.pool.Put(conn)

	exprsJSON, err := json.Marshal(tp.Exprs)
	if err != nil {
		return err
	}

	err = sqlitex.Execute(conn, `
		INSERT INTO topics (user_id, name, exprs, updated)
		VALUES (NULL, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		ON CONFLICT(user_id, name) DO UPDATE SET
			exprs = excluded.exprs,
			updated = excluded.updated
	`, &sqlitex.ExecOptions{
		Args: []interface{}{tp.Name, string(exprsJSON)},
	})

	return err
}

// assembleTopic sets TopicName, ExprIndex and ExprId for each item
func (h *TopicHandler) assembleTopic(name string, exprs []topic.TopicExpr) topic.Topic {
	for index := range exprs {
		for idx := range exprs[index] {
			exprs[index][idx].TopicName = name
			exprs[index][idx].ExprIndex = index
			exprs[index][idx].ExprId = exprs[index].String()
		}
	}
	return topic.Topic{
		Name:  name,
		Exprs: exprs,
	}
}
