package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dawitdargie/perfinsight/sdk"
	_ "github.com/lib/pq"
)

var tracedDB *sdk.TracedDB

// ensureDemoSchema creates and seeds the orders/items tables the demo's
// /orders handler depends on. These tables are specific to the testapp demo
// — they're unrelated to perfinsight's own traces/queries/metrics tables,
// which the collector manages separately.
func ensureDemoSchema(db *sql.DB) error {
	stmts := []string{
		`DROP TABLE IF EXISTS items`,
		`DROP TABLE IF EXISTS orders`,
		`CREATE TABLE orders (id SERIAL PRIMARY KEY)`,
		`CREATE TABLE items (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL REFERENCES orders(id),
			name TEXT NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}

	const seedOrders = 15
	const itemsPerOrder = 2

	for i := 0; i < seedOrders; i++ {
		var orderID int
		if err := db.QueryRow(`INSERT INTO orders DEFAULT VALUES RETURNING id`).Scan(&orderID); err != nil {
			return fmt.Errorf("seed order: %w", err)
		}
		for j := 0; j < itemsPerOrder; j++ {
			name := fmt.Sprintf("item-%d-%d", orderID, j)
			if _, err := db.Exec(`INSERT INTO items (order_id, name) VALUES ($1, $2)`, orderID, name); err != nil {
				return fmt.Errorf("seed item: %w", err)
			}
		}
	}
	return nil
}

func main() {
	sdk.Init("test-service", "http://localhost:9000")

	db, err := sql.Open("postgres", "host=localhost port=5432 user=perfinsight password=perfinsight_secret dbname=perfinsight sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := ensureDemoSchema(db); err != nil {
		log.Fatalf("Demo schema setup failed: %v", err)
	}

	tracedDB = sdk.WrapDB(db)

	http.HandleFunc("/fast", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		fmt.Fprintln(w, "fast response")
	}))

	http.HandleFunc("/orders", sdk.HTTPMiddleware(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		rows, err := tracedDB.QueryContext(ctx, "SELECT id FROM orders")
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

			itemRows, err := tracedDB.QueryContext(ctx, "SELECT name FROM items WHERE order_id = $1", orderID)
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

	srv := &http.Server{Addr: ":8080"}

	go func() {
		fmt.Println("Test app running on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down testapp...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Flush and close the trace exporter last, so any traces from requests
	// handled right up until shutdown still get sent to the collector.
	sdk.Shutdown()

	db.Close()
	log.Println("testapp stopped")
}