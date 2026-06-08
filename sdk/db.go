package sdk

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

type TracedDB struct {
	db *sql.DB
	mu sync.Mutex
}

func WrapDB(db *sql.DB) *TracedDB {
	return &TracedDB{db: db}
}

func (t *TracedDB) recordQuery(sql string, elapsed int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	trace := GetLastTrace()
	if trace == nil {
		return
	}

	for i := range trace.DBQueries {
		if trace.DBQueries[i].SQL == sql {
			trace.DBQueries[i].Count++
			trace.DBQueries[i].Time += elapsed
			trace.DBTime += elapsed
			return
		}
	}

	trace.DBQueries = append(trace.DBQueries, DBQuery{
		SQL:   sql,
		Count: 1,
		Time:  elapsed,
	})
	trace.DBTime += elapsed
}

func (t *TracedDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := t.db.Query(query, args...)
	elapsed := time.Since(start).Milliseconds()
	t.recordQuery(query, elapsed)
	return rows, err
}

func (t *TracedDB) QueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := t.db.QueryRow(query, args...)
	elapsed := time.Since(start).Milliseconds()
	t.recordQuery(query, elapsed)
	return row
}

func (t *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := t.db.QueryContext(ctx, query, args...)
	elapsed := time.Since(start).Milliseconds()
	t.recordQuery(query, elapsed)
	return rows, err
}

func (t *TracedDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := t.db.Exec(query, args...)
	elapsed := time.Since(start).Milliseconds()
	t.recordQuery(query, elapsed)
	return result, err
}

func (t *TracedDB) Close() error {
	return t.db.Close()
}