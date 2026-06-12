package collector

import (
	"database/sql"
	"testing"
)

func TestUpdateBaseline_SkipsZero(t *testing.T) {
	db, err := sql.Open("postgres",
		"host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()
	s := &Storage{db: db}

	// First set a known baseline
	db.Exec(`INSERT INTO metrics (endpoint, request_count, avg_latency, baseline_avg)
		VALUES ('/test-baseline', 1, 100, 100)
		ON CONFLICT (endpoint) DO UPDATE SET baseline_avg = 100`)

	// Try to update with zero — should be skipped
	err = s.UpdateBaseline("/test-baseline", 0)
	if err != nil {
		t.Fatalf("UpdateBaseline failed: %v", err)
	}

	var baseline float64
	db.QueryRow(`SELECT baseline_avg FROM metrics WHERE endpoint = '/test-baseline'`).Scan(&baseline)
	if baseline != 100 {
		t.Errorf("Baseline should remain 100, got %.2f", baseline)
	}

	// Cleanup
	db.Exec(`DELETE FROM metrics WHERE endpoint = '/test-baseline'`)
}

func TestGetHourlyAverage_ReturnsZeroWithNoData(t *testing.T) {
	db, err := sql.Open("postgres",
		"host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()
	s := &Storage{db: db}

	avg, err := s.GetHourlyAverage("/endpoint-that-does-not-exist")
	if err != nil {
		t.Fatalf("GetHourlyAverage failed: %v", err)
	}
	if avg != 0 {
		t.Errorf("Expected 0 for nonexistent endpoint, got %.2f", avg)
	}
}