package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dawitdargie/perfinsight/analysis"
	"github.com/dawitdargie/perfinsight/output"
)

func main() {
	endpoint := flag.String("endpoint", "all", "Endpoint to analyze, or 'all'")
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "host=localhost user=user password=pass dbname=perfinsight sslmode=disable"
	}

	svc, err := analysis.NewAnalysisService(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer svc.Close()

	var endpoints []string
	if *endpoint == "all" {
		endpoints, err = svc.AllEndpoints()
		if err != nil {
			log.Fatalf("Failed to list endpoints: %v", err)
		}
		if len(endpoints) == 0 {
			fmt.Println("No endpoints found. Send some traffic first.")
			return
		}
	} else {
		endpoints = []string{*endpoint}
	}

	for _, ep := range endpoints {
		result, err := svc.AnalyzeEndpoint(ep)
if err != nil {
    fmt.Fprintf(os.Stderr, "Error analyzing %s: %v\n", ep, err)
    continue
}

if result == nil {
    continue
}

fmt.Println(output.FormatResult(result))
	}
}