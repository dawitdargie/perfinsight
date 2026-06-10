package collector

import (
	"testing"

	"github.com/dawitdargie/perfinsight/sdk"
)

func TestValidateTrace_ValidTrace(t *testing.T) {
	trace := sdk.Trace{
		TraceID:      "abc123",
		Endpoint:     "/orders",
		Latency:      100,
		DBTime:       80,
		ExternalTime: 0,
		StatusCode:   200,
	}
	if err := ValidateTrace(trace); err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestValidateTrace_EmptyTraceID(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "",
		Endpoint:   "/orders",
		Latency:    100,
		StatusCode: 200,
	}
	if err := ValidateTrace(trace); err == nil {
		t.Error("Expected error for empty TraceID")
	}
}

func TestValidateTrace_ZeroLatency(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "abc123",
		Endpoint:   "/orders",
		Latency:    0,
		StatusCode: 200,
	}
	if err := ValidateTrace(trace); err == nil {
		t.Error("Expected error for zero Latency")
	}
}

func TestValidateTrace_DBTimeExceedsLatency(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "abc123",
		Endpoint:   "/orders",
		Latency:    100,
		DBTime:     150,
		StatusCode: 200,
	}
	if err := ValidateTrace(trace); err == nil {
		t.Error("Expected error for DBTime > Latency")
	}
}

func TestValidateTrace_InvalidStatusCode(t *testing.T) {
	trace := sdk.Trace{
		TraceID:    "abc123",
		Endpoint:   "/orders",
		Latency:    100,
		StatusCode: 999,
	}
	if err := ValidateTrace(trace); err == nil {
		t.Error("Expected error for invalid status code")
	}
}

func TestValidateBatch_DropsInvalidKeepsValid(t *testing.T) {
	batch := []sdk.Trace{
		{TraceID: "valid-1", Endpoint: "/fast", Latency: 5, StatusCode: 200},
		{TraceID: "", Endpoint: "/orders", Latency: 100, StatusCode: 200},
		{TraceID: "valid-2", Endpoint: "/orders", Latency: 100, StatusCode: 200},
	}
	result := ValidateBatch(batch)
	if len(result) != 2 {
		t.Errorf("Expected 2 valid traces, got %d", len(result))
	}
	if result[0].TraceID != "valid-1" {
		t.Error("First valid trace incorrect")
	}
	if result[1].TraceID != "valid-2" {
		t.Error("Second valid trace incorrect")
	}
}