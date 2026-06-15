package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dawitdargie/perfinsight/analysis"
)

func main() {
	endpoint := flag.String("endpoint", "", "Endpoint to analyze (e.g. /orders)")
	flag.Parse()

	if *endpoint == "" {
		fmt.Println("Usage: analyze -endpoint /orders")
		os.Exit(1)
	}

	svc, err := analysis.NewAnalysisService(
		"host=localhost port=5433 user=user password=pass dbname=perfinsight sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer svc.Close()

	result, err := svc.AnalyzeEndpoint(*endpoint)
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	if result == nil || !result.HasIssues {
		fmt.Printf("No issues detected for %s\n", *endpoint)
		return
	}

	// Raw output for now — Day 22 output layer replaces this.
	fmt.Printf("Endpoint: %s | Analyzed: %s\n", result.Endpoint, result.AnalyzedAt.Format("15:04:05"))
	for _, issue := range result.Issues {
		fmt.Printf("  Pattern: %s | Severity: %s\n", issue.Pattern, issue.Severity)
	}
}