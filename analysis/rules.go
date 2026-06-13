package analysis

import "fmt"

// EvaluateRules runs all detection rules in order and returns matching issues.
// Evaluation order: DB bottleneck → N+1 → External API → Regression.
// Multiple rules can fire simultaneously — this is correct behavior.
func EvaluateRules(input AnalysisInput) []Issue {
	var issues []Issue

	if issue := ruleDBBottleneck(input); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := ruleN1Query(input); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := ruleExternalAPIBottleneck(input); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := rulePerformanceRegression(input); issue != nil {
		issues = append(issues, *issue)
	}

	return issues
}

// ruleDBBottleneck checks whether database time dominates total latency.
func ruleDBBottleneck(input AnalysisInput) *Issue {
	if input.TotalLatency == 0 {
		return nil
	}
	dbRatio := float64(input.DBTime) / float64(input.TotalLatency)
	if dbRatio <= 0.7 {
		return nil
	}
	return &Issue{
		Pattern:    "DATABASE_BOTTLENECK",
		Severity:   "high",
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("DB time: %dms (%.0f%% of total request time)", input.DBTime, dbRatio*100),
			fmt.Sprintf("Total request latency: %dms", input.TotalLatency),
			fmt.Sprintf("Internal processing time: %dms", input.InternalTime),
		},
		Suggestion: []string{
			"Add indexes on frequently queried columns",
			"Reduce SELECT * — select only required columns",
			"Consider caching repeated read queries",
			"Profile slow queries using EXPLAIN ANALYZE",
		},
	}
}

// ruleN1Query detects N+1 query patterns where many queries have the same SQL text.
// Implemented Day 17.
func ruleN1Query(input AnalysisInput) *Issue {
	// Implemented Day 17
	return nil
}

// ruleExternalAPIBottleneck checks whether external API calls dominate total latency.
// Implemented Day 17.
func ruleExternalAPIBottleneck(input AnalysisInput) *Issue {
	// Implemented Day 17
	return nil
}

// rulePerformanceRegression detects significant increases in latency compared to baseline.
// Implemented Day 18.
func rulePerformanceRegression(input AnalysisInput) *Issue {
	// Implemented Day 18
	return nil
}