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

		ctx := InjectTraceID(r.Context(), trace.TraceID)
		r = r.WithContext(ctx)

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		endTime := time.Now()
		latency := endTime.Sub(startTime).Milliseconds()
		trace.Latency = latency
		trace.StatusCode = rw.statusCode

		AddTrace(trace)
		fmt.Printf("[TRACE STORED] ID=%s endpoint=%s latency=%dms status=%d\n", trace.TraceID, trace.Endpoint, trace.Latency, trace.StatusCode)
	}
}