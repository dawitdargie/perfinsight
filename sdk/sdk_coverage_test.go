// sdk/sdk_coverage_test.go
package sdk

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInit_SetsServiceName(t *testing.T) {
	Init("my-test-service", "http://localhost:9000")
	ResetTraces()

	handler := HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	traces := GetTraces()
	if len(traces) == 0 {
		t.Fatal("No trace captured")
	}
	if traces[0].ServiceName != "my-test-service" {
		t.Errorf("Expected service name 'my-test-service', got '%s'", traces[0].ServiceName)
	}
}

func TestFinalizeTrace_RejectsInvalidTrace(t *testing.T) {
	trace := &Trace{
		TraceID:     "test-invalid",
		Endpoint:    "/orders",
		Latency:     100,
		DBTime:      200, // Impossible: DB > Total
		StatusCode:  200,
		ServiceName: "test",
	}

	err := FinalizeTrace(trace)
	if err == nil {
		t.Error("Expected error for DBTime > Latency")
	}
}

func TestHTTPMiddleware_ConcurrentRequestsIsolated(t *testing.T) {
	ResetTraces()

	handler := HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/concurrent", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	traces := GetTraces()
	if len(traces) != 10 {
		t.Errorf("Expected 10 traces, got %d", len(traces))
	}

	seen := make(map[string]bool)
	for _, trace := range traces {
		if seen[trace.TraceID] {
			t.Errorf("Duplicate trace ID: %s", trace.TraceID)
		}
		seen[trace.TraceID] = true
	}
}

func TestContext_ConcurrentRequestsHaveIsolatedIDs(t *testing.T) {
	ids := make(chan string, 10)

	for i := 0; i < 10; i++ {
		go func() {
			traceID := generateTraceID()
			ctx := InjectTraceID(context.Background(), traceID)
			extracted := ExtractTraceID(ctx)
			ids <- extracted
		}()
	}

	collected := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := <-ids
		if collected[id] {
			t.Errorf("Duplicate ID across concurrent contexts: %s", id)
		}
		collected[id] = true
	}
}

func TestResetTraces_ClearsAllTraces(t *testing.T) {
	handler := HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if len(GetTraces()) == 0 {
		t.Skip("No traces to reset")
	}

	ResetTraces()

	if len(GetTraces()) != 0 {
		t.Errorf("Expected 0 traces after reset, got %d", len(GetTraces()))
	}

	if ActiveTraceCount() != 0 {
		t.Errorf("Expected 0 active traces after reset, got %d", ActiveTraceCount())
	}
}