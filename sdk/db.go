package sdk

import (
	"context"
	"database/sql"
	"time"
)

type TracedDB struct {
	db *sql.DB
}

func WrapDB(db *sql.DB) *TracedDB {
	return &TracedDB{db: db}
}

// IMPORTANT: use the *Context methods below and always pass the request's
// context (r.Context(), or a context derived from it) through your handler
// so DB time gets attributed to the correct trace. Under concurrent
// requests, there is no way to know which request a query belongs to
// without this — that's what caused cross-request data corruption before.

func (t *TracedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := t.db.QueryContext(ctx, query, args...)
	elapsed := time.Since(start).Milliseconds()
	recordDBQuery(ExtractTraceID(ctx), query, elapsed)
	return rows, err
}

func (t *TracedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := t.db.QueryRowContext(ctx, query, args...)
	elapsed := time.Since(start).Milliseconds()
	recordDBQuery(ExtractTraceID(ctx), query, elapsed)
	return row
}

func (t *TracedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := t.db.ExecContext(ctx, query, args...)
	elapsed := time.Since(start).Milliseconds()
	recordDBQuery(ExtractTraceID(ctx), query, elapsed)
	return result, err
}

// Query, QueryRow, and Exec are kept only for compatibility with code that
// hasn't migrated to context-aware calls. They talk to the real DB, but
// CANNOT be attributed to any trace (there's no context to read a trace ID
// from), so they will silently NOT show up in DB-time analysis. Migrate
// call sites to the *Context versions above for accurate results.
func (t *TracedDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return t.db.Query(query, args...)
}

func (t *TracedDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return t.db.QueryRow(query, args...)
}

func (t *TracedDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return t.db.Exec(query, args...)
}

func (t *TracedDB) Close() error {
	return t.db.Close()
}