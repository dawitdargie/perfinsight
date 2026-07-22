package sdk

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var currentServiceName string
var globalExporter *Exporter

func Init(serviceName string, collectorURL string) {
	currentServiceName = serviceName
	globalExporter = NewExporter(collectorURL)
}

func HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		traceID := generateTraceID()
		trace := Trace{
			TraceID:  traceID,
			Endpoint: r.URL.Path,
			Method:   r.Method,
		}

		ctx := InjectTraceID(r.Context(), traceID)
		r = r.WithContext(ctx)

		AddTrace(trace)
		defer RemoveTrace(traceID)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		endTime := time.Now()
		latency := endTime.Sub(startTime).Milliseconds()
		if latency <= 0 {
			latency = 1
		}

		t := GetActiveTrace(traceID)
		if t == nil {
			return
		}
		t.Latency = latency
		t.StatusCode = rw.statusCode
		SetServiceName(t, currentServiceName)

		if err := FinalizeTrace(t); err != nil {
			fmt.Fprintf(os.Stderr, "perfinsight: dropping invalid trace %s: %v\n", t.TraceID, err)
			return
		}

		completedTrace := *t
		addCompletedTrace(completedTrace)

		if globalExporter != nil {
			globalExporter.Enqueue(completedTrace)
		}
	}
}

func HTTPMiddlewareHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HTTPMiddleware(next.ServeHTTP)(w, r)
	})
}
// Shutdown flushes any buffered traces and stops the background exporter.
// Call this once, during graceful shutdown, before the process exits —
// otherwise traces enqueued but not yet sent can be lost.
func Shutdown() {
	if globalExporter != nil {
		globalExporter.Close()
	}
}