package sdk

import (
	"fmt"
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

func HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		trace := Trace{
			TraceID:  generateTraceID(),
			Endpoint: r.URL.Path,
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		endTime := time.Now()
		latency := endTime.Sub(startTime).Milliseconds()
		trace.Latency = latency

		fmt.Printf("[TRACE] ID=%s endpoint=%s latency=%dms\n", trace.TraceID, trace.Endpoint, trace.Latency)
	}
}