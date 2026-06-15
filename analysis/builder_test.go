package analysis

import (
	"testing"
)

func TestBuildResult_AssemblesCorrectly(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		DBTime:       280,
		InternalTime: 40,
		BaselineAvg:  100,
		CurrentAvg:   320,
	}
	issues := []Issue{
		{Pattern: "DATABASE_BOTTLENECK", Severity: "high"},
	}
	result := BuildResult(input, issues)
	if result.Endpoint != "/orders" {
		t.Errorf("Expected /orders, got %s", result.Endpoint)
	}
	if result.Latency != 320 {
		t.Errorf("Expected latency 320, got %d", result.Latency)
	}
	if !result.HasIssues {
		t.Error("Expected HasIssues=true")
	}
	if len(result.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(result.Issues))
	}
	if result.AnalyzedAt.IsZero() {
		t.Error("AnalyzedAt should be set")
	}
}

func TestBuildResult_HasIssuesFalseWithNoIssues(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/fast",
		TotalLatency: 5,
	}
	result := BuildResult(input, []Issue{})
	if result.HasIssues {
		t.Error("Expected HasIssues=false with empty issues")
	}
}