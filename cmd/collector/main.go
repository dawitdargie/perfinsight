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

	storage, err := collector.NewStorage("host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		log.Fatalf("Storage init failed: %v", err)
	}
	defer storage.Close()

	pool := collector.NewWorkerPool(srv.TraceBuffer(), 10, storage)
	pool.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Collector running on :9000")
		if err := srv.Start(":9000"); err != http.ErrServerClosed {
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

	// 2. Stop workers — drains remaining batches
	pool.Stop()

	log.Println("Collector stopped")
}
