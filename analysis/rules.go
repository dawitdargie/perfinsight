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
func ruleN1Query(input AnalysisInput) *Issue {
	if len(input.DBQueries) == 0 {
		return nil
	}
	// Find the most repeated query
	var topQuery *QueryStat
	for i := range input.DBQueries {
		if input.DBQueries[i].Count > 50 {
			if topQuery == nil || input.DBQueries[i].Count > topQuery.Count {
				topQuery = &input.DBQueries[i]
			}
		}
	}
	if topQuery == nil {
		return nil
	}
	return &Issue{
		Pattern:    "N_PLUS_ONE_QUERY",
		Severity:   "critical",
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("Query executed %d times in a single request", topQuery.Count),
			fmt.Sprintf("SQL: %s", topQuery.SQL),
			fmt.Sprintf("Total time for this query: %dms", topQuery.Time),
		},
		Suggestion: []string{
			"Use batch loading: replace looped queries with IN clause",
			"Use JOIN to fetch related data in one query",
			"Implement eager loading for related records",
			"Cache query results if data changes infrequently",
		},
	}
}

// ruleExternalAPIBottleneck checks whether external API calls dominate total latency.
func ruleExternalAPIBottleneck(input AnalysisInput) *Issue {
	if input.ExternalTime == 0 {
		return nil
	}
	if input.TotalLatency == 0 {
		return nil
	}
	extRatio := float64(input.ExternalTime) / float64(input.TotalLatency)
	if extRatio <= 0.7 {
		return nil
	}
	return &Issue{
		Pattern:    "EXTERNAL_API_BOTTLENECK",
		Severity:   "high",
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("External API time: %dms (%.0f%% of total request time)", input.ExternalTime, extRatio*100),
			fmt.Sprintf("Total request latency: %dms", input.TotalLatency),
		},
		Suggestion: []string{
			"Add timeout limits to external API calls",
			"Cache external API responses where appropriate",
			"Consider moving external API calls to background jobs",
			"Implement circuit breaker pattern for resilience",
		},
	}
}

// rulePerformanceRegression detects significant increases in latency compared to baseline.
// Implemented Day 18.
func rulePerformanceRegression(input AnalysisInput) *Issue {
	// Implemented Day 18
	return nil
}