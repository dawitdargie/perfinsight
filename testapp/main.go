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
		fmt.Fprintln(w, "orders response")
	}))

	fmt.Println("Test app running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
