package output

import "github.com/dawitdargie/perfinsight/analysis"

const (
	typeRegression  = "REGRESSION"
	typeImprovement = "IMPROVEMENT"
	typeBottleneck  = "BOTTLENECK"
)

// classifyIssue determines the output type for an issue based on its pattern and values.
func classifyIssue(issue analysis.Issue) string {
	if issue.Pattern == "PERFORMANCE_REGRESSION" {
		if issue.CurrentMs > issue.BaselineMs {
			return typeRegression
		}
		return typeImprovement
	}
	return typeBottleneck
}

// severityIcon returns the appropriate emoji icon for a given severity level.
func severityIcon(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return "⚪"
	}
}

// patternTitle returns a human-readable title for a given pattern identifier.
func patternTitle(pattern string) string {
	switch pattern {
	case "DATABASE_BOTTLENECK":
		return "Database Bottleneck"
	case "N_PLUS_ONE_QUERY":
		return "N+1 Query Pattern"
	case "EXTERNAL_API_BOTTLENECK":
		return "External API Bottleneck"
	case "PERFORMANCE_REGRESSION":
		return "Performance Regression"
	default:
		return pattern
	}
}

// patternExplanation returns a one-sentence explanation of why a pattern is a problem.
func patternExplanation(pattern string) string {
	switch pattern {
	case "DATABASE_BOTTLENECK":
		return "Database operations are consuming the majority of this request's time."
	case "N_PLUS_ONE_QUERY":
		return "The same query is being executed repeatedly in a loop instead of using a single batch query."
	case "EXTERNAL_API_BOTTLENECK":
		return "A third-party API call is consuming the majority of this request's time."
	case "PERFORMANCE_REGRESSION":
		return "This endpoint has become significantly slower compared to its historical baseline."
	default:
		return ""
	}
}
