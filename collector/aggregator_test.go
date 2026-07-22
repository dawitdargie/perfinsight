// collector/aggregator_test.go
package collector

import (
	"database/sql"
	"testing"
)

func testAggregatorDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres",
		"host=localhost port=5432 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("PostgreSQL not reachable: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestUpdateBaseline_SkipsZero(t *testing.T) {
	db := testAggregatorDB(t)
	s := &Storage{db: db}
	key := EndpointKey{ServiceName: "test-service", Endpoint: "/test-baseline"}

	_, err := db.Exec(`INSERT INTO metrics (service_name, endpoint, request_count, avg_latency, baseline_avg)
		VALUES ($1, $2, 1, 100, 100)
		ON CONFLICT (service_name, endpoint) DO UPDATE SET baseline_avg = 100`,
		key.ServiceName, key.Endpoint)
	if err != nil {
		t.Fatalf("Setup INSERT failed: %v", err)
	}

	err = s.UpdateBaseline(key, 0)
	if err != nil {
		t.Fatalf("UpdateBaseline failed: %v", err)
	}

	var baseline float64
	db.QueryRow(`SELECT baseline_avg FROM metrics WHERE service_name = $1 AND endpoint = $2`,
		key.ServiceName, key.Endpoint).Scan(&baseline)
	if baseline != 100 {
		t.Errorf("Baseline should remain 100, got %.2f", baseline)
	}

	db.Exec(`DELETE FROM metrics WHERE service_name = $1 AND endpoint = $2`, key.ServiceName, key.Endpoint)
}

func TestGetHourlyAverage_ReturnsZeroWithNoData(t *testing.T) {
	db := testAggregatorDB(t)
	s := &Storage{db: db}
	key := EndpointKey{ServiceName: "test-service", Endpoint: "/endpoint-that-does-not-exist"}

	avg, err := s.GetHourlyAverage(key)
	if err != nil {
		t.Fatalf("GetHourlyAverage failed: %v", err)
	}
	if avg != 0 {
		t.Errorf("Expected 0 for nonexistent endpoint, got %.2f", avg)
	}
}

func TestBaselines_ScopedByService(t *testing.T) {
	db := testAggregatorDB(t)
	s := &Storage{db: db}
	keyA := EndpointKey{ServiceName: "service-a", Endpoint: "/shared-path"}
	keyB := EndpointKey{ServiceName: "service-b", Endpoint: "/shared-path"}

	// Seed a metrics row for each key first (UpdateBaseline only updates an
	// existing row, it doesn't insert one).
	_, err := db.Exec(`INSERT INTO metrics (service_name, endpoint, request_count, avg_latency, baseline_avg)
		VALUES ($1, $2, 1, 0, 0)
		ON CONFLICT (service_name, endpoint) DO NOTHING`,
		keyA.ServiceName, keyA.Endpoint)
	if err != nil {
		t.Fatalf("Setup INSERT A failed: %v", err)
	}
	_, err = db.Exec(`INSERT INTO metrics (service_name, endpoint, request_count, avg_latency, baseline_avg)
		VALUES ($1, $2, 1, 0, 0)
		ON CONFLICT (service_name, endpoint) DO NOTHING`,
		keyB.ServiceName, keyB.Endpoint)
	if err != nil {
		t.Fatalf("Setup INSERT B failed: %v", err)
	}

	if err := s.UpdateBaseline(keyA, 50); err != nil {
		t.Fatalf("UpdateBaseline A failed: %v", err)
	}
	if err := s.UpdateBaseline(keyB, 900); err != nil {
		t.Fatalf("UpdateBaseline B failed: %v", err)
	}

	var baselineA, baselineB float64
	db.QueryRow(`SELECT baseline_avg FROM metrics WHERE service_name = $1 AND endpoint = $2`,
		keyA.ServiceName, keyA.Endpoint).Scan(&baselineA)
	db.QueryRow(`SELECT baseline_avg FROM metrics WHERE service_name = $1 AND endpoint = $2`,
		keyB.ServiceName, keyB.Endpoint).Scan(&baselineB)

	if baselineA != 50 {
		t.Errorf("Expected baseline A = 50, got %.2f", baselineA)
	}
	if baselineB != 900 {
		t.Errorf("Expected baseline B = 900, got %.2f", baselineB)
	}
	if baselineA == baselineB {
		t.Errorf("Expected different baselines for same endpoint path under different services, both were %.2f", baselineA)
	}

	db.Exec(`DELETE FROM metrics WHERE service_name = $1 AND endpoint = $2`, keyA.ServiceName, keyA.Endpoint)
	db.Exec(`DELETE FROM metrics WHERE service_name = $1 AND endpoint = $2`, keyB.ServiceName, keyB.Endpoint)
}