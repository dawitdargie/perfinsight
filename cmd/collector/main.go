package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dawitdargie/perfinsight/collector"
)

func main() {
	srv := collector.NewServer()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost port=5432 user=user password=pass dbname=perfinsight sslmode=disable"
	}

	storage, err := collector.NewStorage(dbURL)
	if err != nil {
		log.Fatalf("Storage init failed: %v", err)
	}
	defer storage.Close()

	pool := collector.NewWorkerPool(srv.TraceBuffer(), 10, storage)
	pool.Start()

	aggregator := collector.NewAggregator(storage, 60*time.Second)
	aggregator.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		port := os.Getenv("PORT")
		if port == "" {
			port = "9000"
		}
		log.Printf("Collector running on :%s", port)
		if err := srv.Start(":" + port); err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("Shutting down...")

	// 1. Stop accepting new requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	// 2. Stop aggregator
	aggregator.Stop()

	// 3. Stop workers — drains remaining batches
	pool.Stop()

	log.Println("Collector stopped")
}
