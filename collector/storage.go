package collector

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/dawitdargie/perfinsight/sdk"
)

// EndpointKey identifies a unique (service, endpoint) pair.
type EndpointKey struct {
	ServiceName string
	Endpoint    string
}

type Storage struct {
	db *sql.DB
}

func NewStorage(databaseURL string) (*Storage, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	s := &Storage{db: db}
	if err := s.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}
	return s, nil
}

func (s *Storage) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS traces (
			trace_id TEXT PRIMARY KEY,
			endpoint TEXT NOT NULL,
			method TEXT NOT NULL DEFAULT 'GET',
			total_latency INTEGER NOT NULL,
			db_time INTEGER NOT NULL DEFAULT 0,
			external_time INTEGER NOT NULL DEFAULT 0,
			internal_time INTEGER NOT NULL DEFAULT 0,
			status_code INTEGER NOT NULL,
			service_name TEXT NOT NULL DEFAULT 'unknown',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS queries (
			id SERIAL PRIMARY KEY,
			trace_id TEXT NOT NULL REFERENCES traces(trace_id),
			sql_text TEXT NOT NULL,
			execution_count INTEGER NOT NULL DEFAULT 1,
			total_time INTEGER NOT NULL DEFAULT 0
		)`,
		// NOTE: metrics is now keyed by (service_name, endpoint) instead of
		// endpoint alone. If you're migrating an existing database, DROP the
		// old metrics table first (see chat) — CREATE TABLE IF NOT EXISTS
		// will not alter an existing table's schema.
		`CREATE TABLE IF NOT EXISTS metrics (
			service_name TEXT NOT NULL,
			endpoint TEXT NOT NULL,
			request_count INTEGER NOT NULL DEFAULT 0,
			error_count INTEGER NOT NULL DEFAULT 0,
			avg_latency FLOAT NOT NULL DEFAULT 0,
			baseline_avg FLOAT NOT NULL DEFAULT 0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (service_name, endpoint)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_endpoint ON traces(endpoint)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_service_endpoint ON traces(service_name, endpoint)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_created_at ON traces(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_queries_trace_id ON queries(trace_id)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec: %w", err)
		}
	}
	return nil
}

func (s *Storage) Save(t sdk.Trace) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO traces (trace_id, endpoint, method, total_latency, db_time,
			external_time, internal_time, status_code, service_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (trace_id) DO NOTHING`,
		t.TraceID, t.Endpoint, t.Method, t.Latency, t.DBTime,
		t.ExternalTime, t.InternalTime, t.StatusCode, t.ServiceName, t.Timestamp)
	if err != nil {
		return fmt.Errorf("insert trace: %w", err)
	}

	for _, q := range t.DBQueries {
		_, err = tx.Exec(`
			INSERT INTO queries (trace_id, sql_text, execution_count, total_time)
			VALUES ($1, $2, $3, $4)`,
			t.TraceID, q.SQL, q.Count, q.Time)
		if err != nil {
			return fmt.Errorf("insert query: %w", err)
		}
	}

	if err := s.updateMetrics(tx, t); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Storage) updateMetrics(tx *sql.Tx, t sdk.Trace) error {
	isError := t.StatusCode >= 400
	errorIncrement := 0
	if isError {
		errorIncrement = 1
	}
	_, err := tx.Exec(`
		INSERT INTO metrics (service_name, endpoint, request_count, error_count, avg_latency, baseline_avg)
		VALUES ($1, $2, 1, $3, $4, $4)
		ON CONFLICT (service_name, endpoint) DO UPDATE SET
			request_count = metrics.request_count + 1,
			error_count = metrics.error_count + $3,
			avg_latency = (metrics.avg_latency * metrics.request_count + $4) / (metrics.request_count + 1),
			updated_at = NOW()`,
		t.ServiceName, t.Endpoint, errorIncrement, float64(t.Latency))
	return err
}

func (s *Storage) GetEndpoints() ([]EndpointKey, error) {
	rows, err := s.db.Query(`SELECT service_name, endpoint FROM metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []EndpointKey
	for rows.Next() {
		var k EndpointKey
		if err := rows.Scan(&k.ServiceName, &k.Endpoint); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *Storage) GetHourlyAverage(key EndpointKey) (float64, error) {
	var avg float64
	err := s.db.QueryRow(`
		SELECT COALESCE(AVG(total_latency), 0)
		FROM traces
		WHERE endpoint = $1 AND service_name = $2
		AND created_at > NOW() - INTERVAL '1 hour'
	`, key.Endpoint, key.ServiceName).Scan(&avg)
	return avg, err
}

func (s *Storage) UpdateBaseline(key EndpointKey, baseline float64) error {
	if baseline == 0 {
		return nil
	}
	_, err := s.db.Exec(`
		UPDATE metrics
		SET baseline_avg = $1,
			updated_at = NOW()
		WHERE service_name = $2 AND endpoint = $3
	`, baseline, key.ServiceName, key.Endpoint)
	return err
}

func (s *Storage) GetRecentAverage(key EndpointKey) (float64, error) {
	var avg float64
	err := s.db.QueryRow(`
		SELECT COALESCE(AVG(total_latency), 0)
		FROM traces
		WHERE endpoint = $1 AND service_name = $2
		AND created_at > NOW() - INTERVAL '5 minutes'
	`, key.Endpoint, key.ServiceName).Scan(&avg)
	return avg, err
}

func (s *Storage) Close() error {
	return s.db.Close()
}