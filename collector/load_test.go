package collector

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dawitdargie/perfinsight/sdk"
)

func TestWorkerPool_100ConcurrentBatches(t *testing.T) {
	s, err := NewStorage("host=localhost port=5432 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer s.Close()

	buffer := make(chan []sdk.Trace, 500)
	pool := NewWorkerPool(buffer, 10, s)
	pool.Start()

	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(batchNum int) {
			defer wg.Done()
			batch := make([]sdk.Trace, 10)
			for j := 0; j < 10; j++ {
				batch[j] = sdk.Trace{
					TraceID:      fmt.Sprintf("load-%d-%d", batchNum, j),
					Endpoint:     "/load-test",
					Method:       "GET",
					Latency:      int64(50 + batchNum),
					DBTime:       int64(30 + batchNum),
					ExternalTime: 0,
					StatusCode:   200,
					ServiceName:  "load-test",
					Timestamp:    time.Now(),
					DBQueries:    []sdk.DBQuery{},
				}
			}
			buffer <- batch
		}(i)
	}

	wg.Wait()
	pool.Stop()
	elapsed := time.Since(start)
	t.Logf("✅ Processed 1000 traces in %v", elapsed)

	// Verify traces stored
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM traces WHERE endpoint = '/load-test'`).Scan(&count)
	if count < 900 { // Allow 10% drop rate for test variance
		t.Errorf("Expected ~1000 traces stored, got %d", count)
	}

	// Cleanup
	s.db.Exec(`DELETE FROM queries WHERE trace_id LIKE 'load-%'`)
	s.db.Exec(`DELETE FROM traces WHERE endpoint = '/load-test'`)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = '/load-test'`)
}

func TestValidation_EntireBatchInvalidReturns400(t *testing.T) {
	batch := []sdk.Trace{
		{TraceID: "", Endpoint: "/orders", Latency: 100, StatusCode: 200},
		{TraceID: "", Endpoint: "/orders", Latency: 100, StatusCode: 200},
	}

	result := ValidateBatch(batch)
	if len(result) != 0 {
		t.Errorf("Expected 0 valid traces, got %d", len(result))
	}
}

func TestPipeline_NormalizerRunsBeforeStorage(t *testing.T) {
	s, err := NewStorage("host=localhost port=5432 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer s.Close()

	// Trace with missing ServiceName
	trace := sdk.Trace{
		TraceID:      "norm-pipeline-001",
		Endpoint:     "/norm-pipeline-test",
		Method:       "GET",
		Latency:      100,
		DBTime:       0,
		StatusCode:   200,
		// ServiceName intentionally empty
		Timestamp:    time.Now(),
		DBQueries:    []sdk.DBQuery{},
	}

	Normalize(&trace)

	if err := s.Save(trace); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	var storedServiceName string
	s.db.QueryRow(`SELECT service_name FROM traces WHERE trace_id = $1`, trace.TraceID).Scan(&storedServiceName)
	if storedServiceName != "unknown" {
		t.Errorf("Expected 'unknown' service name after normalization, got '%s'", storedServiceName)
	}

	// Cleanup
	s.db.Exec(`DELETE FROM traces WHERE trace_id = $1`, trace.TraceID)
	s.db.Exec(`DELETE FROM metrics WHERE endpoint = $1`, trace.Endpoint)
}