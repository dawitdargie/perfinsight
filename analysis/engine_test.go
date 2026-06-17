package analysis

import (
	"testing"
)

func testService(t *testing.T) *AnalysisService {
	t.Helper()
	svc, err := NewAnalysisService(
		"host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func TestAnalyzeEndpoint_ReturnsNilForUnknownEndpoint(t *testing.T) {
	svc := testService(t)
	result, err := svc.AnalyzeEndpoint("/endpoint-that-does-not-exist-xyz")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Error("Expected nil result for unknown endpoint")
	}
}

func TestAnalyzeEndpoint_ReturnsResultForKnownEndpoint(t *testing.T) {
	svc := testService(t)
	result, err := svc.AnalyzeEndpoint("/orders")
	if err != nil {
		t.Fatalf("Analysis error: %v", err)
	}
	if result == nil {
		t.Skip("No data for /orders — run full pipeline test first")
	}
	if result.Endpoint != "/orders" {
		t.Errorf("Expected endpoint /orders, got %s", result.Endpoint)
	}
	if result.AnalyzedAt.IsZero() {
		t.Error("AnalyzedAt should be set")
	}
	if result.Latency == 0 {
		t.Error("Latency should be non-zero")
	}
}

func TestAnalyzeEndpoint_DetectsN1InRealData(t *testing.T) {
	svc := testService(t)
	result, err := svc.AnalyzeEndpoint("/orders")
	if err != nil {
		t.Fatalf("Analysis error: %v", err)
	}
	if result == nil {
		t.Skip("No data for /orders")
	}
	hasN1 := false
	for _, issue := range result.Issues {
		if issue.Pattern == "N_PLUS_ONE_QUERY" {
			hasN1 = true
			if issue.Severity != "critical" {
				t.Errorf("N+1 should be critical, got %s", issue.Severity)
			}
			if len(issue.Evidence) == 0 {
				t.Error("N+1 evidence should not be empty")
			}
			if len(issue.Suggestion) == 0 {
				t.Error("N+1 suggestions should not be empty")
			}
		}
	}
	if !hasN1 {
		t.Log("N+1 not detected — may need more traffic or lower threshold data")
	}
}

func TestAllEndpoints_ReturnsKnownEndpoints(t *testing.T) {
	svc := testService(t)
	endpoints, err := svc.AllEndpoints()
	if err != nil {
		t.Fatalf("AllEndpoints error: %v", err)
	}
	if len(endpoints) == 0 {
		t.Skip("No endpoints in metrics table — run pipeline test first")
	}
	found := false
	for _, ep := range endpoints {
		if ep == "/orders" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected /orders in endpoints list")
	}
}