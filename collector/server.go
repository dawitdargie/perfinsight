package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dawitdargie/perfinsight/sdk"
)

type Server struct {
	traceBuffer chan []sdk.Trace
	httpServer  *http.Server
}

func NewServer() *Server {
	return &Server{
		traceBuffer: make(chan []sdk.Trace, 500),
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

	select {
	case s.traceBuffer <- traces:
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

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ingest-trace", s.handleIngestTrace)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}