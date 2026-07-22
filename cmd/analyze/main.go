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
	serviceName := flag.String("service", "", "Service name to analyze (required unless -endpoint=all)")
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

	var targets []analysis.EndpointKey
	if *endpoint == "all" {
		targets, err = svc.AllEndpoints(*serviceName)
		if err != nil {
			log.Fatalf("Failed to list endpoints: %v", err)
		}
		if len(targets) == 0 {
			fmt.Println("No endpoints found. Send some traffic first.")
			return
		}
	} else {
		if *serviceName == "" {
			log.Fatal("-service is required when -endpoint is set (two different projects can share an endpoint path)")
		}
		targets = []analysis.EndpointKey{{ServiceName: *serviceName, Endpoint: *endpoint}}
	}

	for _, key := range targets {
		result, err := svc.AnalyzeEndpoint(key.ServiceName, key.Endpoint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing %s [%s]: %v\n", key.Endpoint, key.ServiceName, err)
			continue
		}
		if result == nil {
			continue
		}
		fmt.Println(output.FormatResult(result))
	}
}