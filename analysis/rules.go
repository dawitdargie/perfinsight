package analysis

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
// Implemented Day 16.
func ruleDBBottleneck(input AnalysisInput) *Issue {
	// Implemented Day 16
	return nil
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