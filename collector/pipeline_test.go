package collector

import (
	"fmt"
	"testing"
	"time"

	"github.com/dawitdargie/perfinsight/sdk"
)

func testStorage(t *testing.T) *Storage {
	t.Helper()
	s, err := NewStorage("host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("Storage init failed: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPipeline_SingleTraceStoredCorrectly(t *testing.T) {
	s := testStorage(t)
	trace := sdk.Trace{
		TraceID:      "pipe-test-001",
		Endpoint:     "/pipeline-test",
		Method:       "GET",
		Latency:      150,
		DBTime:       120,
		ExternalTime: 0,
		InternalTime: 30,
		StatusCode:   200,
		ServiceName:  "test-service",
		Timestamp:    time.Now(),
		DBQueries: []sdk.DBQuery{
			{SQL: "SELECT id FROM orders", Count: 1, Time: 20},
			{SQL: "SELECT name FROM items WHERE order_id = $1", Count: 5, Time: 100},
		},
	}
	Normalize(&trace)
	if err := s.Save(trace); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	var storedLatency, storedDBTime, storedInternal int64
	err := s.db.QueryRow(`
		SELECT total_latency, db_time, internal_time
		FROM traces WHERE trace_id = $1
	`, trace.TraceID).Scan(&storedLatency, &storedDBTime, &storedInternal)
	if err != nil {
		t.Fatalf("Trace not found: %v", err)
	}
	if storedLatency != 150 {
		t.Errorf("Expected latency 150, got %d", storedLatency)
	}
	if storedDBTime != 120 {
		t.Errorf("Expected DBTime 120, got %d", storedDBTime)
	}
	if storedInternal != 30 {
		t.Errorf("Expected InternalTime 30, got %d", storedInternal)
	}

	s.db.Exec(`DELETE FROM queries WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM traces WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, trace.Endpoint)
}

func TestPipeline_QueriesStoredWithCounts(t *testing.T) {
	s := testStorage(t)
	trace := sdk.Trace{
		TraceID:    "pipe-test-002",
		Endpoint:   "/query-count-test",
		Method:     "GET",
		Latency:    100,
		DBTime:     90,
		StatusCode: 200,
		ServiceName: "test-service",
		Timestamp:  time.Now(),
		DBQueries: []sdk.DBQuery{
			{SQL: "SELECT id FROM orders", Count: 1, Time: 10},
			{SQL: "SELECT * FROM items WHERE order_id = $1", Count: 50, Time: 80},
		},
	}
	Normalize(&trace)
	if err := s.Save(trace); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	rows, err := s.db.Query(`
		SELECT sql_text, execution_count
		FROM queries WHERE trace_id = $1
		ORDER BY execution_count DESC
	`, trace.TraceID)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	type queryRow struct {
		sql   string
		count int
	}
	var results []queryRow
	for rows.Next() {
		var qr queryRow
		rows.Scan(&qr.sql, &qr.count)
		results = append(results, qr)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 query rows, got %d", len(results))
	}
	if results[0].count != 50 {
		t.Errorf("Expected max count 50, got %d", results[0].count)
	}
	if results[1].count != 1 {
		t.Errorf("Expected count 1, got %d", results[1].count)
	}

	s.db.Exec(`DELETE FROM queries WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM traces WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, trace.Endpoint)
}

func TestPipeline_MetricsAvgLatencyUpdates(t *testing.T) {
	s := testStorage(t)
	endpoint := "/metrics-avg-test"

	s.db.Exec(`DELETE FROM queries WHERE trace_id LIKE 'avg-test-%'`)
	s.db.Exec(`DELETE FROM traces WHERE endpoint = $1`, endpoint)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, endpoint)

	for i, latency := range []int64{100, 200, 300} {
		trace := sdk.Trace{
			TraceID:    fmt.Sprintf("avg-test-%d", i),
			Endpoint:   endpoint,
			Method:     "GET",
			Latency:    latency,
			DBTime:     0,
			StatusCode: 200,
			ServiceName: "test",
			Timestamp:  time.Now(),
			DBQueries:  []sdk.DBQuery{},
		}
		Normalize(&trace)
		if err := s.Save(trace); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	var requestCount int
	var avgLatency float64
	err := s.db.QueryRow(`
		SELECT request_count, avg_latency
		FROM metrics WHERE endpoint = $1
	`, endpoint).Scan(&requestCount, &avgLatency)
	if err != nil {
		t.Fatalf("Metrics query failed: %v", err)
	}
	if requestCount != 3 {
		t.Errorf("Expected request_count=3, got %d", requestCount)
	}
	if avgLatency < 195 || avgLatency > 205 {
		t.Errorf("Expected avg ~200ms, got %.2f", avgLatency)
	}

	s.db.Exec(`DELETE FROM queries WHERE trace_id LIKE 'avg-test-%'`)
	s.db.Exec(`DELETE FROM traces WHERE endpoint = $1`, endpoint)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, endpoint)
}

func TestPipeline_DuplicateTraceIDHandled(t *testing.T) {
	s := testStorage(t)
	trace := sdk.Trace{
		TraceID:    "duplicate-test-001",
		Endpoint:   "/dup-test",
		Method:     "GET",
		Latency:    100,
		DBTime:     0,
		StatusCode: 200,
		ServiceName: "test",
		Timestamp:  time.Now(),
		DBQueries:  []sdk.DBQuery{},
	}
	Normalize(&trace)

	if err := s.Save(trace); err != nil {
		t.Fatalf("First save failed: %v", err)
	}
	if err := s.Save(trace); err != nil {
		t.Fatalf("Second save (duplicate) failed: %v", err)
	}

	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM traces WHERE trace_id = $1`, trace.TraceID).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 trace row, got %d", count)
	}

	s.db.Exec(`DELETE FROM queries WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM traces WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, trace.Endpoint)
}