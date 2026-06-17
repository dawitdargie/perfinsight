package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dawitdargie/perfinsight/analysis"
)

func main() {
	endpoint := flag.String("endpoint", "all", "Endpoint to analyze, or 'all'")
	flag.Parse()

	svc, err := analysis.NewAnalysisService(
		"host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
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
		if result == nil || !result.HasIssues {
			fmt.Printf("[%s] No issues detected\n", ep)
			continue
		}
		// Raw output — Day 22 replaces this
		fmt.Printf("\n[%s] %d issue(s) found:\n", ep, len(result.Issues))
		for _, issue := range result.Issues {
			fmt.Printf(" - %s (%s)\n", issue.Pattern, issue.Severity)
		}
	}
}