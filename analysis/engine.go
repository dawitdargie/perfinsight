package analysis

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// AnalysisService holds the database connection for the analysis engine.
// It is completely separate from the collector's connection pool.
type AnalysisService struct {
	db *sql.DB
}

// NewAnalysisService creates a new AnalysisService with its own database connection.
func NewAnalysisService(databaseURL string) (*AnalysisService, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(3)
	return &AnalysisService{db: db}, nil
}

// Close closes the database connection.
func (as *AnalysisService) Close() error {
	return as.db.Close()
}

// buildInput queries the database and assembles an AnalysisInput for the given endpoint.
// Returns (nil, nil) if no data exists for the endpoint.
func (as *AnalysisService) buildInput(endpoint string) (*AnalysisInput, error) {
	// Query 1 — Get latest trace for the endpoint
	var totalLatency, dbTime, externalTime, internalTime int64
	err := as.db.QueryRow(`
		SELECT total_latency, db_time, external_time, internal_time
		FROM traces
		WHERE endpoint = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, endpoint).Scan(&totalLatency, &dbTime, &externalTime, &internalTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest trace: %w", err)
	}

	// Query 2 — Get baseline average from metrics table
	var baselineAvg float64
	err = as.db.QueryRow(`
		SELECT baseline_avg FROM metrics WHERE endpoint = $1
	`, endpoint).Scan(&baselineAvg)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query baseline: %w", err)
	}

	// Query 3 — Get current 5-minute average
	var currentAvg float64
	err = as.db.QueryRow(`
		SELECT COALESCE(AVG(total_latency), 0)
		FROM traces
		WHERE endpoint = $1
		AND created_at > NOW() - INTERVAL '5 minutes'
	`, endpoint).Scan(&currentAvg)
	if err != nil {
		return nil, fmt.Errorf("query current avg: %w", err)
	}

	// Query 4 — Get queries for the latest trace
	rows, err := as.db.Query(`
		SELECT q.sql_text, q.execution_count, q.total_time
		FROM queries q
		JOIN traces t ON q.trace_id = t.trace_id
		WHERE t.endpoint = $1
		ORDER BY t.created_at DESC
		LIMIT 50
	`, endpoint)
	if err != nil {
		return nil, fmt.Errorf("query queries: %w", err)
	}
	defer rows.Close()

	var dbQueries []QueryStat
	for rows.Next() {
		var q QueryStat
		if err := rows.Scan(&q.SQL, &q.Count, &q.Time); err != nil {
			return nil, fmt.Errorf("scan query row: %w", err)
		}
		dbQueries = append(dbQueries, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate queries: %w", err)
	}

	return &AnalysisInput{
		Endpoint:     endpoint,
		TotalLatency: totalLatency,
		DBTime:       dbTime,
		ExternalTime: externalTime,
		InternalTime: internalTime,
		BaselineAvg:  baselineAvg,
		CurrentAvg:   currentAvg,
		DBQueries:    dbQueries,
	}, nil
}

// AnalyzeEndpoint analyzes the given endpoint and returns a list of detected issues.
func (as *AnalysisService) AnalyzeEndpoint(endpoint string) ([]Issue, error) {
	input, err := as.buildInput(endpoint)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, nil // No data yet
	}
	issues := EvaluateRules(*input)
	return issues, nil
}