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
	if issue := ruleHighErrorRate(input); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := ruleHighInternalProcessing(input); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := ruleHighLatency(input); issue != nil {
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

// ruleHighErrorRate detects endpoints with elevated error rates.
func ruleHighErrorRate(input AnalysisInput) *Issue {
	if input.RequestCount == 0 {
		return nil
	}
	if input.ErrorRate <= 5.0 {
		return nil
	}
	severity := "high"
	if input.ErrorRate > 20 {
		severity = "critical"
	}
	return &Issue{
		Pattern:    "HIGH_ERROR_RATE",
		Severity:   severity,
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("Error rate: %.1f%% (%d errors out of %d requests)", input.ErrorRate, input.ErrorCount, input.RequestCount),
			fmt.Sprintf("Endpoint: %s", input.Endpoint),
		},
		Suggestion: []string{
			"Check for recent code changes or deployments that may have introduced bugs",
			"Review application logs for stack traces and error messages",
			"Verify upstream dependencies are healthy",
			"Add more granular error tracking to identify specific failure modes",
		},
	}
}

// ruleHighInternalProcessing detects when application logic dominates request time.
func ruleHighInternalProcessing(input AnalysisInput) *Issue {
	if input.TotalLatency < 100 {
		return nil
	}
	internalRatio := float64(input.InternalTime) / float64(input.TotalLatency)
	if internalRatio <= 0.5 {
		return nil
	}
	severity := "medium"
	if internalRatio > 0.7 {
		severity = "high"
	}
	if internalRatio > 0.85 {
		severity = "critical"
	}
	return &Issue{
		Pattern:    "HIGH_INTERNAL_PROCESSING",
		Severity:   severity,
		Confidence: "medium",
		Evidence: []string{
			fmt.Sprintf("Internal processing time: %dms (%.0f%% of total request time)", input.InternalTime, internalRatio*100),
			fmt.Sprintf("Total request latency: %dms", input.TotalLatency),
			fmt.Sprintf("DB time: %dms, External time: %dms", input.DBTime, input.ExternalTime),
		},
		Suggestion: []string{
			"Profile CPU usage during request processing",
			"Review serialization/deserialization logic for efficiency",
			"Check for inefficient loops or data transformations",
			"Consider using worker pools for CPU-intensive operations",
		},
	}
}

// ruleHighLatency detects endpoints with high absolute latency.
// Uses both a minimum threshold and baseline comparison to avoid false positives.
func ruleHighLatency(input AnalysisInput) *Issue {
	if input.TotalLatency < 500 {
		return nil
	}
	// If baseline exists, require latency > 1.5x baseline
	if input.BaselineAvg > 0 {
		ratio := float64(input.TotalLatency) / input.BaselineAvg
		if ratio <= 1.5 {
			return nil
		}
	}
	severity := "medium"
	if input.TotalLatency >= 2000 {
		severity = "high"
	}
	return &Issue{
		Pattern:    "HIGH_LATENCY",
		Severity:   severity,
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("Total latency: %dms", input.TotalLatency),
			fmt.Sprintf("Endpoint: %s", input.Endpoint),
			fmt.Sprintf("DB time: %dms, Internal time: %dms", input.DBTime, input.InternalTime),
		},
		Suggestion: []string{
			"Consider adding caching for frequently accessed data",
			"Evaluate if synchronous processing can be moved to async/background jobs",
			"Review endpoint logic for unnecessary computation or blocking calls",
			"Consider pagination or streaming for large payloads",
		},
	}
}

// ruleN1Query detects N+1 query patterns by identifying repeated SQL in a single request.
// Detection is based on count alone — severity reflects the actual performance impact.
func ruleN1Query(input AnalysisInput) *Issue {
	if len(input.DBQueries) == 0 {
		return nil
	}

	// Find the most repeated query
	var topQuery *QueryStat
	for i := range input.DBQueries {
		if topQuery == nil || input.DBQueries[i].Count > topQuery.Count {
			topQuery = &input.DBQueries[i]
		}
	}
	if topQuery == nil || topQuery.Count < 10 {
		return nil
	}

	severity := "medium"
	if topQuery.Count >= 50 {
		severity = "high"
	}
	if topQuery.Count >= 200 || input.TotalLatency > 500 || input.DBTime > 300 {
		severity = "critical"
	}

	return &Issue{
		Pattern:    "N_PLUS_ONE_QUERY",
		Severity:   severity,
		Confidence: "high",
		Evidence: []string{
			fmt.Sprintf("Query executed %d times in a single request", topQuery.Count),
			fmt.Sprintf("SQL: %s", topQuery.SQL),
			fmt.Sprintf("Total time for this query: %dms", topQuery.Time),
			fmt.Sprintf("DB time: %dms", input.DBTime),
			fmt.Sprintf("Total request latency: %dms", input.TotalLatency),
		},
		Suggestion: []string{
			"Replace looped queries with batch loading (IN clause)",
			"Use eager loading or JOINs",
			"Cache repeated lookups where appropriate",
			"Review ORM query generation",
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
