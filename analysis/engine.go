package analysis

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// analysisWindow is the default lookback window for hot detection.
// Traces older than this are not checked for active issues.
const analysisWindow = "1 hour"

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
	// Query 1 — Get the most recent trace in the analysis window
	var totalLatency, dbTime, externalTime, internalTime int64
	var serviceName string
	err := as.db.QueryRow(`
		SELECT total_latency, db_time, external_time, internal_time, service_name
		FROM traces
		WHERE endpoint = $1
		AND created_at > NOW() - INTERVAL '`+analysisWindow+`'
		AND total_latency > 0
		ORDER BY created_at DESC
        LIMIT 1
	`, endpoint).Scan(&totalLatency, &dbTime, &externalTime, &internalTime, &serviceName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query worst trace: %w", err)
	}

	// Query 2 — Get baseline average from metrics table
	var baselineAvg float64
	err = as.db.QueryRow(`
		SELECT baseline_avg FROM metrics WHERE endpoint = $1
	`, endpoint).Scan(&baselineAvg)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query baseline: %w", err)
	}

	// Query 3 — Get current average within the analysis window
	var currentAvg float64
	err = as.db.QueryRow(`
		SELECT COALESCE(AVG(total_latency), 0)
		FROM traces
		WHERE endpoint = $1
		AND created_at > NOW() - INTERVAL '`+analysisWindow+`'
	`, endpoint).Scan(&currentAvg)
	if err != nil {
		return nil, fmt.Errorf("query current avg: %w", err)
	}

	// Query 4 — Get error rate from metrics table
	var errorCount, requestCount int
	err = as.db.QueryRow(`
		SELECT COALESCE(error_count, 0), COALESCE(request_count, 0)
		FROM metrics WHERE endpoint = $1
	`, endpoint).Scan(&errorCount, &requestCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query metrics: %w", err)
	}

	// Query 5 — Get queries aggregated by SQL across the analysis window
	rows, err := as.db.Query(`
		SELECT q.sql_text, MAX(q.execution_count), COALESCE(AVG(q.total_time), 0)::bigint
		FROM queries q
		JOIN traces t ON q.trace_id = t.trace_id
		WHERE t.endpoint = $1
		AND t.created_at > NOW() - INTERVAL '`+analysisWindow+`'
		GROUP BY q.sql_text
		ORDER BY MAX(q.execution_count) DESC
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
		ServiceName:  serviceName,
		Endpoint:     endpoint,
		TotalLatency: totalLatency,
		DBTime:       dbTime,
		ExternalTime: externalTime,
		InternalTime: internalTime,
		BaselineAvg:  baselineAvg,
		CurrentAvg:   currentAvg,
		DBQueries:    dbQueries,
		ErrorCount:   errorCount,
		RequestCount: requestCount,
		ErrorRate:    computeErrorRate(errorCount, requestCount),
	}, nil
}

// AnalyzeEndpoint analyzes the given endpoint and returns a structured result.
func (as *AnalysisService) AnalyzeEndpoint(endpoint string) (*Result, error) {
	input, err := as.buildInput(endpoint)
	if err != nil {
		return nil, err
	}
	if input == nil {
		return nil, nil // No data yet
	}
	issues := EvaluateRules(*input)
	result := BuildResult(*input, issues)
	return result, nil
}

// computeErrorRate returns the error rate as a percentage.
func computeErrorRate(errors, requests int) float64 {
	if requests == 0 {
		return 0
	}
	return float64(errors) / float64(requests) * 100
}

// AllEndpoints returns all known endpoints from the metrics table.
func (as *AnalysisService) AllEndpoints() ([]string, error) {
	rows, err := as.db.Query(`
		SELECT DISTINCT endpoint
		FROM traces
		WHERE created_at > NOW() - INTERVAL '1 hour'
		ORDER BY endpoint
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var endpoints []string

	for rows.Next() {
		var ep string
		if err := rows.Scan(&ep); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}
