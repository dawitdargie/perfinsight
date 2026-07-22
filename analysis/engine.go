package analysis

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

const analysisWindow = "1 hour"

type AnalysisService struct {
	db *sql.DB
}

func NewAnalysisService(databaseURL string) (*AnalysisService, error) {
	connStr := databaseURL
	if !strings.Contains(connStr, "binary_parameters") {
		if strings.Contains(connStr, "?") {
			connStr += "&binary_parameters=yes"
		} else {
			connStr += "?binary_parameters=yes"
		}
	}
	db, err := sql.Open("postgres", connStr)
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

func (as *AnalysisService) Close() error {
	return as.db.Close()
}

// buildInput now takes serviceName explicitly — every query is scoped to
// (service_name, endpoint), so two different projects sharing an endpoint
// path never contaminate each other's results.
func (as *AnalysisService) buildInput(serviceName, endpoint string) (*AnalysisInput, error) {
	var totalLatency, dbTime, externalTime, internalTime int64
	err := as.db.QueryRow(`
		SELECT total_latency,
		       COALESCE(db_time, 0),
		       COALESCE(external_time, 0),
		       COALESCE(internal_time, 0)
		FROM traces
		WHERE endpoint = $1 AND service_name = $2
		AND created_at > NOW() - INTERVAL '`+analysisWindow+`'
		AND total_latency > 0
		ORDER BY created_at DESC
		LIMIT 1
	`, endpoint, serviceName).Scan(&totalLatency, &dbTime, &externalTime, &internalTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query worst trace: %w", err)
	}

	var baselineAvg float64
	err = as.db.QueryRow(`
		SELECT baseline_avg FROM metrics WHERE endpoint = $1 AND service_name = $2
	`, endpoint, serviceName).Scan(&baselineAvg)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query baseline: %w", err)
	}

	var currentAvg float64
	err = as.db.QueryRow(`
		SELECT COALESCE(AVG(total_latency), 0)
		FROM traces
		WHERE endpoint = $1 AND service_name = $2
		AND created_at > NOW() - INTERVAL '`+analysisWindow+`'
	`, endpoint, serviceName).Scan(&currentAvg)
	if err != nil {
		return nil, fmt.Errorf("query current avg: %w", err)
	}

	var errorCount, requestCount int
	err = as.db.QueryRow(`
		SELECT COALESCE(error_count, 0), COALESCE(request_count, 0)
		FROM metrics WHERE endpoint = $1 AND service_name = $2
	`, endpoint, serviceName).Scan(&errorCount, &requestCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("query metrics: %w", err)
	}

	rows, err := as.db.Query(`
		SELECT q.sql_text, COALESCE(MAX(q.execution_count), 0), COALESCE(AVG(q.total_time), 0)::bigint
		FROM queries q
		JOIN traces t ON q.trace_id = t.trace_id
		WHERE t.endpoint = $1 AND t.service_name = $2
		AND t.created_at > NOW() - INTERVAL '`+analysisWindow+`'
		GROUP BY q.sql_text
		ORDER BY COALESCE(MAX(q.execution_count), 0) DESC
		LIMIT 50
	`, endpoint, serviceName)
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

func (as *AnalysisService) AnalyzeEndpoint(serviceName, endpoint string) (*Result, error) {
	input, err := as.buildInput(serviceName, endpoint)
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

func computeErrorRate(errors, requests int) float64 {
	if requests == 0 {
		return 0
	}
	return float64(errors) / float64(requests) * 100
}

// AllEndpoints returns all known (service, endpoint) pairs. Pass an empty
// serviceName to list across all services; pass a specific name to scope to
// one project.
func (as *AnalysisService) AllEndpoints(serviceName string) ([]EndpointKey, error) {
	query := `
		SELECT DISTINCT service_name, endpoint
		FROM traces
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`
	var args []interface{}
	if serviceName != "" {
		query += ` AND service_name = $1`
		args = append(args, serviceName)
	}
	query += ` ORDER BY service_name, endpoint`

	rows, err := as.db.Query(query, args...)
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