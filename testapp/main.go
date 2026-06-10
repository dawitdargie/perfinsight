package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dawitdargie/perfinsight/sdk"
	_ "github.com/lib/pq"
)

var tracedDB *sdk.TracedDB

func main() {
	sdk.Init("test-service", "http://localhost:9000")

	db, err := sql.Open("postgres", "host=localhost port=5433 user=user password=pass dbname=perftest sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	tracedDB = sdk.WrapDB(db)

	http.HandleFunc("/fast", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "fast response")
	}))

	http.HandleFunc("/orders", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		rows, err := tracedDB.Query("SELECT id FROM orders")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var orderID int
			if err := rows.Scan(&orderID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			itemRows, err := tracedDB.Query("SELECT name FROM items WHERE order_id = $1", orderID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for itemRows.Next() {
				var name string
				if err := itemRows.Scan(&name); err != nil {
					itemRows.Close()
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				fmt.Fprintf(w, "order %d: %s\n", orderID, name)
			}
			itemRows.Close()
		}
	}))

	http.HandleFunc("/missing", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "not found")
	}))

	http.HandleFunc("/error", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "error")
	}))

	http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		traces := sdk.GetTraces()
		for _, t := range traces {
			fmt.Fprintf(w, "Endpoint: %s\n", t.Endpoint)
			fmt.Fprintf(w, "ServiceName: %s\n", t.ServiceName)
			fmt.Fprintf(w, "Latency: %dms\n", t.Latency)
			fmt.Fprintf(w, "DBTime: %dms\n", t.DBTime)
			fmt.Fprintf(w, "ExternalTime: %dms\n", t.ExternalTime)
			fmt.Fprintf(w, "InternalTime: %dms\n", t.InternalTime)
			fmt.Fprintf(w, "Timestamp: %s\n", t.Timestamp.Format(time.RFC3339Nano))
			fmt.Fprintf(w, "Queries: %d\n\n", len(t.DBQueries))
			for _, q := range t.DBQueries {
				fmt.Fprintf(w, "  SQL: %s\n", q.SQL)
				fmt.Fprintf(w, "  Count: %d\n", q.Count)
				fmt.Fprintf(w, "  Time: %dms\n\n", q.Time)
			}
		}
	})

	// Note: exporter.Close() would be called via OS signal handler in production.
	// For now, traces flush on the 5-second ticker.
	fmt.Println("Test app running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
