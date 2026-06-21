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

// ruleN1Query detects N+1 query patterns using evidence gates.
// An N+1 is only reported when:
//  1. A SQL statement is repeated more than once in a single request
//  2. Database time dominates the request (>70% of latency)
//  3. There is measurable performance impact (slow request OR significant DB time)
func ruleN1Query(input AnalysisInput) *Issue {
	if len(input.DBQueries) == 0 || input.TotalLatency == 0 {
		return nil
	}

	// Find the most repeated query (highest execution count per trace)
	var topQuery *QueryStat
	for i := range input.DBQueries {
		if input.DBQueries[i].Count < 2 {
			continue // single execution is not a repeated pattern
		}
		if topQuery == nil || input.DBQueries[i].Count > topQuery.Count {
			topQuery = &input.DBQueries[i]
		}
	}
	if topQuery == nil {
		return nil // no SQL was repeated more than once
	}

	dbRatio := float64(input.DBTime) / float64(input.TotalLatency)

	// Evidence Gate 1: DB must dominate the request
	if dbRatio <= 0.7 {
		return nil // repeated query exists but DB is not the bottleneck
	}

	// Evidence Gate 2: There must be measurable performance impact
	// Either the request is slow (>200ms) OR the repeated queries consumed significant DB time (>100ms)
	if input.TotalLatency <= 200 && input.DBTime <= 100 {
		return nil // repeated pattern exists but has no meaningful performance impact
	}

	// Severity derived from impact, not arbitrary count thresholds
	var severity string
	if input.TotalLatency > 200 && input.DBTime > 100 {
		severity = "critical"
	} else {
		severity = "high"
	}

	return &Issue{
		Pattern:    "N_PLUS_ONE_QUERY",
		Severity:   severity,
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("Query executed %d times in a single request", topQuery.Count),
			fmt.Sprintf("SQL: %s", topQuery.SQL),
			fmt.Sprintf("Total time for this query: %dms", topQuery.Time),
			fmt.Sprintf("DB time: %dms (%.0f%% of total request time)", input.DBTime, dbRatio*100),
			fmt.Sprintf("Total request latency: %dms", input.TotalLatency),
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
func rulePerformanceRegression(input AnalysisInput) *Issue {
	if input.BaselineAvg == 0 {
		return nil // No baseline yet
	}
	if input.CurrentAvg == 0 {
		return nil // No recent data
	}
	ratio := input.CurrentAvg / input.BaselineAvg
	if ratio <= 2.0 {
		return nil
	}
	percentIncrease := (ratio - 1.0) * 100
	delta := input.CurrentAvg - input.BaselineAvg
	return &Issue{
		Pattern:    "PERFORMANCE_REGRESSION",
		Severity:   "critical",
		Confidence: "high",
		BaselineMs: input.BaselineAvg,
		CurrentMs:  input.CurrentAvg,
		Evidence: []string{
			fmt.Sprintf("Current avg: %.0fms (last 5 minutes)", input.CurrentAvg),
			fmt.Sprintf("Baseline avg: %.0fms (last 1 hour)", input.BaselineAvg),
			fmt.Sprintf("Increase: %.1fx slower (+%.0f%% / +%.0fms)", ratio, percentIncrease, delta),
		},
		Suggestion: []string{
			"Review recent code changes or deployments",
			"Check for new or modified database queries",
			"Look for changes in external API call patterns",
			"Profile the endpoint to identify the source of slowdown",
		},
	}
}
