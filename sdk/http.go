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

var currentServiceName string

func Init(serviceName string) {
	currentServiceName = serviceName
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
		}
		mu.Unlock()

		fmt.Printf("[TRACE STORED] ID=%s endpoint=%s latency=%dms status=%d\n", trace.TraceID, trace.Endpoint, latency, rw.statusCode)
	}
}