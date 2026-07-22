//output/full_output_test.go
package output

import (
	"strings"
	"testing"
	"time"

	"github.com/dawitdargie/perfinsight/analysis"
)

func TestFullOutput_CompleteIssueSet(t *testing.T) {
	result := &analysis.Result{
		Endpoint:     "/orders",
		AnalyzedAt:   time.Now(),
		Latency:      320,
		DBTime:       280,
		InternalTime: 40,
		BaselineAvg:  100,
		CurrentAvg:   320,
		HasIssues:    true,
		Issues: []analysis.Issue{
			{
				Pattern:    "DATABASE_BOTTLENECK",
				Severity:   "high",
				Confidence: "high",
				Evidence: []string{
					"DB time: 280ms (87% of total request time)",
					"Total request latency: 320ms",
				},
				BaselineMs: 0,
				CurrentMs:  0,
			},
			{
				Pattern:    "N_PLUS_ONE_QUERY",
				Severity:   "critical",
				Confidence: "high",
				Evidence: []string{
					"Query executed 55 times in a single request",
					"SQL: SELECT * FROM items WHERE order_id = $1",
				},
				BaselineMs: 0,
				CurrentMs:  0,
			},
			{
				Pattern:    "PERFORMANCE_REGRESSION",
				Severity:   "critical",
				Confidence: "high",
				Evidence: []string{
					"Current avg: 320ms (last 5 minutes)",
					"Baseline avg: 100ms (last 1 hour)",
				},
				BaselineMs: 100,
				CurrentMs:  320,
			},
		},
	}

	out := FormatResult(result)

	// Verify all required sections present
	requiredSections := []struct {
		name string
		text string
	}{
		{"header", "Performance Analysis: /orders"},
		{"db bottleneck icon", "🟠"},
		{"n+1 icon", "🔴"},
		{"regression icon", "🔴"},
		{"db title", "Database Bottleneck"},
		{"n+1 title", "N+1 Query Pattern"},
		{"regression title", "Performance Regression"},
		{"evidence header", "Evidence:"},
		{"fix header", "Suggested fixes:"},
		{"before/after", "slower than usual"},
		{"summary", "issue(s) detected"},
		{"timestamp", "Analyzed at:"},
		{"separator", "─"},
	}

	for _, section := range requiredSections {
		if !strings.Contains(out, section.text) {
			t.Errorf("Missing section '%s' (expected '%s')", section.name, section.text)
		}
	}
}

func TestFullOutput_ImprovementCase(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/checkout",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "PERFORMANCE_REGRESSION",
				Severity:   "critical",
				Evidence:   []string{"System is faster"},
				BaselineMs: 200,
				CurrentMs:  140, // Faster — improvement
			},
		},
	}

	out := FormatResult(result)

	if !strings.Contains(out, "faster") {
		t.Error("Expected 'faster' in improvement output")
	}
}