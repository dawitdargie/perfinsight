package output

import (
	"strings"
	"testing"

	"github.com/dawitdargie/perfinsight/analysis"
)

func TestFormatResult_NoIssuesReturnsCleanMessage(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/fast",
		HasIssues: false,
	}
	output := FormatResult(result)
	if !strings.Contains(output, "No performance issues") {
		t.Error("Expected clean message for no issues")
	}
	if !strings.Contains(output, "/fast") {
		t.Error("Expected endpoint name in output")
	}
}

func TestFormatResult_NilResultHandledSafely(t *testing.T) {
	output := FormatResult(nil)
	if output == "" {
		t.Error("Expected non-empty output for nil result")
	}
}

func TestFormatResult_HeaderContainsEndpoint(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		Latency:   320,
		DBTime:    280,
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "DATABASE_BOTTLENECK",
				Severity:   "high",
				Evidence:   []string{"DB time: 280ms (87%)"},
				Suggestion: []string{"Add indexes"},
			},
		},
	}
	output := FormatResult(result)
	if !strings.Contains(output, "/orders") {
		t.Error("Expected endpoint in output")
	}
	if !strings.Contains(output, "320") {
		t.Error("Expected latency in output")
	}
}

func TestFormatResult_SeverityIconPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{"DB slow"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "🟠") {
		t.Error("Expected orange icon for high severity")
	}
}

func TestFormatResult_CriticalSeverityShowsRedIcon(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "N_PLUS_ONE_QUERY",
				Severity: "critical",
				Evidence: []string{"Query repeated 150x"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "🔴") {
		t.Error("Expected red icon for critical severity")
	}
}

func TestFormatResult_EvidenceSectionPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{"DB time: 280ms (87% of total)"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Evidence") {
		t.Error("Expected Evidence section")
	}
	if !strings.Contains(out, "DB time: 280ms") {
		t.Error("Expected evidence content")
	}
}

func TestFormatResult_MultipleIssuesSeparated(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{Pattern: "DATABASE_BOTTLENECK", Severity: "high", Evidence: []string{"DB slow"}},
			{Pattern: "N_PLUS_ONE_QUERY", Severity: "critical", Evidence: []string{"Query 150x"}},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Database Bottleneck") {
		t.Error("Expected first issue title")
	}
	if !strings.Contains(out, "N+1 Query Pattern") {
		t.Error("Expected second issue title")
	}
	if !strings.Contains(out, "─") {
		t.Error("Expected separator between issues")
	}
}