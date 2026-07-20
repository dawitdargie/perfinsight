package sdk

import (
	"net/http"
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

		trace := Trace{
			TraceID:  generateTraceID(),
			Endpoint: r.URL.Path,
		}

		ctx := InjectTraceID(r.Context(), trace.TraceID)
		r = r.WithContext(ctx)

		AddTrace(trace)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		endTime := time.Now()
		latency := endTime.Sub(startTime).Milliseconds()

		var completedTrace Trace
		validTrace := false

		mu.Lock()
		if len(traces) > 0 {
			last := &traces[len(traces)-1]
			last.Latency = latency
			last.StatusCode = rw.statusCode
			SetServiceName(last, currentServiceName)

			if err := FinalizeTrace(last); err != nil {
				traces = traces[:len(traces)-1]
				mu.Unlock()
				return
			}

			completedTrace = *last
			validTrace = true
		}
		mu.Unlock()

		if validTrace && globalExporter != nil {
			globalExporter.Enqueue(completedTrace)
		}
	}
}

// HTTPMiddlewareHandler wraps any http.Handler.
// This allows users to instrument the entire application/router.
func HTTPMiddlewareHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		HTTPMiddleware(next.ServeHTTP)(w, r)
	})
}