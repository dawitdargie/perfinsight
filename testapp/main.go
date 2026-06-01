package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/dawitdargie/perfinsight/sdk"
)

func main() {
	http.HandleFunc("/fast", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fast response")
	}))

	http.HandleFunc("/orders", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		traceID := sdk.ExtractTraceID(r.Context())
		fmt.Printf("[HANDLER] Trace ID available in handler: %s\n", traceID)
		fmt.Fprintln(w, "orders response")
	}))

	http.HandleFunc("/missing", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "not found")
	}))

	http.HandleFunc("/error", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "error")
	}))

	fmt.Println("Test app running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
