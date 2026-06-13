package analysis

import (
	"strings"
	"testing"
)

func TestRuleDBBottleneck_FiresAbove70Percent(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		DBTime:       280, // 87.5%
		ExternalTime: 0,
		InternalTime: 40,
	}
	issue := ruleDBBottleneck(input)
	if issue == nil {
		t.Fatal("Expected issue, got nil")
	}
	if issue.Pattern != "DATABASE_BOTTLENECK" {
		t.Errorf("Expected DATABASE_BOTTLENECK, got %s", issue.Pattern)
	}
	if issue.Severity != "high" {
		t.Errorf("Expected severity high, got %s", issue.Severity)
	}
	if len(issue.Evidence) == 0 {
		t.Error("Expected evidence, got none")
	}
	if len(issue.Suggestion) == 0 {
		t.Error("Expected suggestions, got none")
	}
}

func TestRuleDBBottleneck_DoesNotFireAtExactly70Percent(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 100,
		DBTime:       70, // Exactly 70%
		ExternalTime: 0,
		InternalTime: 30,
	}
	issue := ruleDBBottleneck(input)
	if issue != nil {
		t.Error("Expected nil at exactly 70%, got issue")
	}
}

func TestRuleDBBottleneck_DoesNotFireBelow70Percent(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 100,
		DBTime:       50, // 50%
		ExternalTime: 0,
		InternalTime: 50,
	}
	issue := ruleDBBottleneck(input)
	if issue != nil {
		t.Errorf("Expected nil, got issue: %s", issue.Pattern)
	}
}

func TestRuleDBBottleneck_HandlesZeroLatency(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 0,
		DBTime:       0,
	}
	// Must not panic
	issue := ruleDBBottleneck(input)
	if issue != nil {
		t.Error("Expected nil for zero latency")
	}
}

func TestRuleDBBottleneck_EvidenceContainsPercentage(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		DBTime:       280,
		InternalTime: 40,
	}
	issue := ruleDBBottleneck(input)
	if issue == nil {
		t.Fatal("Expected issue")
	}
	found := false
	for _, e := range issue.Evidence {
		if strings.Contains(e, "%") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Evidence should contain percentage value")
	}
}