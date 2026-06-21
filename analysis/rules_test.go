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

func TestRuleN1Query_FiresWhenDBDominatesAndImpact(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 500,
		DBTime:       450, // 90% DB ratio
		DBQueries: []QueryStat{
			{SQL: "SELECT id FROM orders", Count: 1, Time: 10},
			{SQL: "SELECT * FROM items WHERE order_id = $1", Count: 55, Time: 440},
		},
	}
	issue := ruleN1Query(input)
	if issue == nil {
		t.Fatal("Expected N+1 issue, got nil")
	}
	if issue.Pattern != "N_PLUS_ONE_QUERY" {
		t.Errorf("Expected N_PLUS_ONE_QUERY, got %s", issue.Pattern)
	}
	if issue.Severity != "critical" {
		t.Errorf("Expected critical severity, got %s", issue.Severity)
	}
}

func TestRuleN1Query_DoesNotFireWhenDBNotDominant(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 500,
		DBTime:       200, // 40% DB ratio — below 70% gate
		DBQueries: []QueryStat{
			{SQL: "SELECT * FROM items", Count: 55, Time: 200},
		},
	}
	issue := ruleN1Query(input)
	if issue != nil {
		t.Error("Expected no N+1 issue when DB is not dominant")
	}
}

func TestRuleN1Query_DoesNotFireWhenLowImpact(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 50,  // fast request
		DBTime:       45,  // 90% DB ratio, but only 45ms
		DBQueries: []QueryStat{
			{SQL: "SELECT * FROM items", Count: 5, Time: 40},
		},
	}
	issue := ruleN1Query(input)
	if issue != nil {
		t.Error("Expected no N+1 issue for fast request with low impact")
	}
}

func TestRuleN1Query_ReportsMostRepeatedQuery(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 500,
		DBTime:       480, // 96% DB ratio
		DBQueries: []QueryStat{
			{SQL: "SELECT * FROM items", Count: 5, Time: 60},
			{SQL: "SELECT * FROM tags", Count: 120, Time: 420},
		},
	}
	issue := ruleN1Query(input)
	if issue == nil {
		t.Fatal("Expected issue")
	}
	if !strings.Contains(issue.Evidence[0], "120") {
		t.Error("Expected most repeated query (count=120) to be reported")
	}
}

func TestRuleN1Query_ReturnsNilWithNoQueries(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/fast",
		TotalLatency: 5,
		DBQueries:    []QueryStat{},
	}
	issue := ruleN1Query(input)
	if issue != nil {
		t.Error("Expected nil with no queries")
	}
}

func TestRuleExternalAPIBottleneck_FiresAbove70Percent(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/checkout",
		TotalLatency: 100,
		DBTime:       10,
		ExternalTime: 80, // 80%
		InternalTime: 10,
	}
	issue := ruleExternalAPIBottleneck(input)
	if issue == nil {
		t.Fatal("Expected issue, got nil")
	}
	if issue.Pattern != "EXTERNAL_API_BOTTLENECK" {
		t.Errorf("Expected EXTERNAL_API_BOTTLENECK, got %s", issue.Pattern)
	}
}

func TestRuleExternalAPIBottleneck_ReturnsNilWhenZero(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 100,
		DBTime:       80,
		ExternalTime: 0,
		InternalTime: 20,
	}
	issue := ruleExternalAPIBottleneck(input)
	if issue != nil {
		t.Error("Expected nil when ExternalTime is zero")
	}
}

func TestRuleExternalAPIBottleneck_DoesNotFireBelow70Percent(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/checkout",
		TotalLatency: 100,
		ExternalTime: 60, // 60%
	}
	issue := ruleExternalAPIBottleneck(input)
	if issue != nil {
		t.Error("Expected nil below 70%")
	}
}

func TestRuleRegression_FiresWhenMoreThan2xSlower(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		BaselineAvg:  100,
		CurrentAvg:   320, // 3.2x
	}
	issue := rulePerformanceRegression(input)
	if issue == nil {
		t.Fatal("Expected issue, got nil")
	}
	if issue.Pattern != "PERFORMANCE_REGRESSION" {
		t.Errorf("Expected PERFORMANCE_REGRESSION, got %s", issue.Pattern)
	}
	if issue.Severity != "critical" {
		t.Errorf("Expected critical, got %s", issue.Severity)
	}
}

func TestRuleRegression_DoesNotFireAtExactly2x(t *testing.T) {
	input := AnalysisInput{
		Endpoint:    "/orders",
		BaselineAvg: 100,
		CurrentAvg:  200, // Exactly 2x
	}
	issue := rulePerformanceRegression(input)
	if issue != nil {
		t.Error("Expected nil at exactly 2x")
	}
}

func TestRuleRegression_DoesNotFireBelow2x(t *testing.T) {
	input := AnalysisInput{
		Endpoint:    "/orders",
		BaselineAvg: 100,
		CurrentAvg:  150, // 1.5x
	}
	issue := rulePerformanceRegression(input)
	if issue != nil {
		t.Errorf("Expected nil, got issue: %s", issue.Pattern)
	}
}

func TestRuleRegression_ReturnsNilWithNoBaseline(t *testing.T) {
	input := AnalysisInput{
		Endpoint:    "/new-endpoint",
		BaselineAvg: 0, // No baseline
		CurrentAvg:  500,
	}
	issue := rulePerformanceRegression(input)
	if issue != nil {
		t.Error("Expected nil with no baseline")
	}
}

func TestRuleRegression_SetsBaselineAndCurrentMs(t *testing.T) {
	input := AnalysisInput{
		Endpoint:    "/orders",
		BaselineAvg: 100,
		CurrentAvg:  320,
	}
	issue := rulePerformanceRegression(input)
	if issue == nil {
		t.Fatal("Expected issue")
	}
	if issue.BaselineMs != 100 {
		t.Errorf("Expected BaselineMs=100, got %.2f", issue.BaselineMs)
	}
	if issue.CurrentMs != 320 {
		t.Errorf("Expected CurrentMs=320, got %.2f", issue.CurrentMs)
	}
}

func TestRuleRegression_EvidenceContainsMultiplier(t *testing.T) {
	input := AnalysisInput{
		Endpoint:    "/orders",
		BaselineAvg: 100,
		CurrentAvg:  320,
	}
	issue := rulePerformanceRegression(input)
	if issue == nil {
		t.Fatal("Expected issue")
	}
	found := false
	for _, e := range issue.Evidence {
		if strings.Contains(e, "3.2") {
			found = true
			break
		}
	}
	if !found {
		t.Error("Evidence should contain 3.2x multiplier")
	}
}
