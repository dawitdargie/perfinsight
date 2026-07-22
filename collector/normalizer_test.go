//collector/normalizer_test.go
package collector

import (
	"testing"

	"github.com/dawitdargie/perfinsight/sdk"
)

func TestNormalize_DefaultServiceName(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "n-001",
		Endpoint:   "/orders",
		Latency:    100,
		StatusCode: 200,
	}
	Normalize(&trace)
	if trace.ServiceName != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", trace.ServiceName)
	}
}

func TestNormalize_DefaultTimestamp(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "n-002",
		Endpoint:   "/orders",
		Latency:    100,
		StatusCode: 200,
	}
	Normalize(&trace)
	if trace.Timestamp.IsZero() {
		t.Error("Timestamp should be set by normalizer")
	}
}

func TestNormalize_RecomputesInternalTime(t *testing.T) {
	trace := sdk.Trace{
		TraceID:      "n-003",
		Endpoint:     "/orders",
		Latency:      100,
		DBTime:       70,
		ExternalTime: 10,
		InternalTime: 999, // Wrong value from SDK
		StatusCode:   200,
	}
	Normalize(&trace)
	// InternalTime = 100 - 70 - 10 = 20
	if trace.InternalTime != 20 {
		t.Errorf("Expected 20, got %d", trace.InternalTime)
	}
}

func TestNormalize_InternalTimeNeverNegative(t *testing.T) {
	trace := sdk.Trace{
		TraceID:      "n-004",
		Endpoint:     "/orders",
		Latency:      100,
		DBTime:       90,
		ExternalTime: 20, // Sum > Latency
		StatusCode:   200,
	}
	Normalize(&trace)
	if trace.InternalTime < 0 {
		t.Errorf("InternalTime should be >= 0, got %d", trace.InternalTime)
	}
}

func TestNormalize_InitializesDBQueries(t *testing.T) {
	trace := sdk.Trace{
		TraceID:   "n-005",
		Endpoint:  "/fast",
		Latency:   5,
		StatusCode: 200,
		DBQueries: nil,
	}
	Normalize(&trace)
	if trace.DBQueries == nil {
		t.Error("DBQueries should be initialized to empty slice")
	}
}