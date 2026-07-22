// collector/server.go
package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dawitdargie/perfinsight/analysis"
	"github.com/dawitdargie/perfinsight/output"
	"github.com/dawitdargie/perfinsight/sdk"
)

type Server struct {
	traceBuffer chan []sdk.Trace
	httpServer  *http.Server
	dbURL       string
}

func NewServer(dbURL string) *Server {
	return &Server{
		traceBuffer: make(chan []sdk.Trace, 500),
		dbURL:       dbURL,
	}
}

func (s *Server) handleIngestTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var traces []sdk.Trace
	if err := json.Unmarshal(body, &traces); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	validTraces := ValidateBatch(traces)
	if len(validTraces) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "no valid traces in batch")
		return
	}

	select {
	case s.traceBuffer <- validTraces:
	default:
		http.Error(w, "buffer full", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		http.Error(w, "service query parameter is required. Usage: ?endpoint=all&service=YOUR_SERVICE_NAME", http.StatusBadRequest)
		return
	}

	endpoint := r.URL.Query().Get("endpoint")
	if endpoint == "" {
		endpoint = "all"
	}

	svc, err := analysis.NewAnalysisService(s.dbURL)
	if err != nil {
		http.Error(w, "analysis service error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer svc.Close()

	if endpoint == "all" {
		targets, err := svc.RecentEndpoints(serviceName, 3)
		if err != nil {
			http.Error(w, "list endpoints error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if len(targets) == 0 {
			fmt.Fprintln(w, "No endpoints found. Send some traffic first.")
			return
		}
		for _, key := range targets {
			result, err := svc.AnalyzeEndpoint(key.ServiceName, key.Endpoint)
			if err != nil {
				fmt.Fprintf(w, "Error analyzing %s [%s]: %v\n\n", key.Endpoint, key.ServiceName, err)
				continue
			}
			if result == nil {
				fmt.Fprintf(w, "No data yet for %s [%s]\n\n", key.Endpoint, key.ServiceName)
				continue
			}
			fmt.Fprint(w, output.FormatResult(result))
			fmt.Fprintln(w)
		}
		return
	}

	// Specific endpoint — check if it was recently accessed
	recent, err := svc.IsEndpointRecent(serviceName, endpoint)
	if err != nil {
		http.Error(w, "check endpoint error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !recent {
		fmt.Fprintf(w, "Endpoint %s has not been accessed recently. Send traffic to %s and re-run the analysis.\n", endpoint, endpoint)
		return
	}

	key := analysis.EndpointKey{ServiceName: serviceName, Endpoint: endpoint}
	result, err := svc.AnalyzeEndpoint(key.ServiceName, key.Endpoint)
	if err != nil {
		fmt.Fprintf(w, "Error analyzing %s [%s]: %v\n", key.Endpoint, key.ServiceName, err)
		return
	}
	if result == nil {
		fmt.Fprintf(w, "No data yet for %s [%s]\n", key.Endpoint, key.ServiceName)
		return
	}
	fmt.Fprint(w, output.FormatResult(result))
	fmt.Fprintln(w)
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ingest-trace", s.handleIngestTrace)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/analyze", s.handleAnalyze)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) TraceBuffer() chan []sdk.Trace {
	return s.traceBuffer
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
