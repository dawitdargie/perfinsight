//analysis/combined_test.go
package analysis

import "testing"

func TestEvaluateRules_DBBottleneckAndN1FireSimultaneously(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		DBTime:       280, // 87.5% → fires DB rule
		ExternalTime: 0,
		InternalTime: 40,
		DBQueries: []QueryStat{
			{SQL: "SELECT id FROM orders", Count: 1, Time: 10},
			{SQL: "SELECT * FROM items WHERE order_id = $1", Count: 55, Time: 270},
		},
		BaselineAvg: 0,
		CurrentAvg:  0,
	}

	issues := EvaluateRules(input)

	hasDB := false
	hasN1 := false
	for _, issue := range issues {
		if issue.Pattern == "DATABASE_BOTTLENECK" {
			hasDB = true
		}
		if issue.Pattern == "N_PLUS_ONE_QUERY" {
			hasN1 = true
		}
	}

	if !hasDB {
		t.Error("Expected DATABASE_BOTTLENECK")
	}
	if !hasN1 {
		t.Error("Expected N_PLUS_ONE_QUERY")
	}
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}
}

func TestEvaluateRules_CleanInputProducesZeroIssues(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/fast",
		TotalLatency: 5,
		DBTime:       2, // 40% — below 70%
		ExternalTime: 0,
		InternalTime: 2, // 40% — below 50%
		DBQueries:    []QueryStat{},
		BaselineAvg:  4,
		CurrentAvg:   5, // 1.25x — below 2x
	}

	issues := EvaluateRules(input)
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for clean input, got %d: %v", len(issues), issues)
	}
}

func TestEvaluateRules_RegressionAndN1FireSimultaneously(t *testing.T) {
	input := AnalysisInput{
		Endpoint:     "/orders",
		TotalLatency: 320,
		DBTime:       280,
		ExternalTime: 0,
		InternalTime: 40,
		DBQueries: []QueryStat{
			{SQL: "SELECT * FROM items WHERE order_id = $1", Count: 55, Time: 270},
		},
		BaselineAvg: 100,
		CurrentAvg:  320, // 3.2x → fires regression
	}

	issues := EvaluateRules(input)

	patterns := make(map[string]bool)
	for _, issue := range issues {
		patterns[issue.Pattern] = true
	}

	if !patterns["N_PLUS_ONE_QUERY"] {
		t.Error("Expected N_PLUS_ONE_QUERY")
	}
	if !patterns["PERFORMANCE_REGRESSION"] {
		t.Error("Expected PERFORMANCE_REGRESSION")
	}
}