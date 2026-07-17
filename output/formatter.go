package output

import (
	"fmt"
	"math"
	"strings"

	"github.com/dawitdargie/perfinsight/analysis"
	"github.com/dawitdargie/perfinsight/output/templates"
)

// FormatResult converts a Result into a complete CLI report string.
// This is the only exported function in the output package.
func FormatResult(result *analysis.Result) string {
	if result == nil {
		return "No analysis result available.\n"
	}
	if !result.HasIssues {
		return fmt.Sprintf("✅ No performance issues detected for %s\n", result.Endpoint)
	}

	var sb strings.Builder
	sb.WriteString(formatHeader(result))
	sb.WriteString("\n")

	for i, issue := range result.Issues {
		sb.WriteString(formatIssue(issue))
		if i < len(result.Issues)-1 {
			sb.WriteString("\n" + strings.Repeat("─", 50) + "\n\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(formatFooter(result))
	return sb.String()
}

func formatHeader(result *analysis.Result) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠️ Performance Analysis: %s\n", result.Endpoint))
	sb.WriteString(strings.Repeat("═", 50) + "\n")
	sb.WriteString(fmt.Sprintf(" Total latency: %dms\n", result.Latency))
	sb.WriteString(fmt.Sprintf(" DB time: %dms\n", result.DBTime))
	sb.WriteString(fmt.Sprintf(" Internal time: %dms\n", result.InternalTime))
	sb.WriteString(fmt.Sprintf(" Issues found: %d\n", len(result.Issues)))
	sb.WriteString(strings.Repeat("═", 50))
	return sb.String()
}

func formatIssue(issue analysis.Issue) string {
	var sb strings.Builder
	icon := severityIcon(issue.Severity)
	title := patternTitle(issue.Pattern)
	explanation := patternExplanation(issue.Pattern)

	sb.WriteString(fmt.Sprintf("\n%s %s\n", icon, title))
	if explanation != "" {
		sb.WriteString(fmt.Sprintf(" %s\n", explanation))
	}
	sb.WriteString("\n")

	// Change section — only for regression/improvement
	if issue.BaselineMs > 0 {
		sb.WriteString(formatChangeSection(issue))
		sb.WriteString("\n")
	}

	// Evidence section
	sb.WriteString(formatEvidenceSection(issue))
	sb.WriteString("\n")

	// Fix section
	sb.WriteString(formatFixSection(issue))
	return sb.String()
}

func formatEvidenceSection(issue analysis.Issue) string {
	if len(issue.Evidence) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("🔍 Evidence:\n")
	for _, e := range issue.Evidence {
		display := e
		if len(display) > 120 {
			display = display[:117] + "..."
		}
		sb.WriteString(fmt.Sprintf(" - %s\n", display))
	}
	return sb.String()
}

func formatChangeSection(issue analysis.Issue) string {
	if issue.BaselineMs == 0 {
		return ""
	}
	m := computeMetrics(issue.BaselineMs, issue.CurrentMs)
	var sb strings.Builder
	sb.WriteString("📊 Change:\n")
	if m.isRegression {
		// Primary: multiplier
		// Secondary: absolute + percentage in parentheses
		sb.WriteString(fmt.Sprintf(" ~%.1f× slower than usual\n", m.multiplier))
		sb.WriteString(fmt.Sprintf(" (%.0fms → %.0fms, ≈ +%.0f%%)\n",
			issue.BaselineMs, issue.CurrentMs, m.percentage))
	} else {
		// Primary: percentage improvement
		// Secondary: multiplier in parentheses
		sb.WriteString(fmt.Sprintf(" ~%.0f%% faster than usual\n",
			math.Abs(m.percentage)))
		sb.WriteString(fmt.Sprintf(" (%.0fms → %.0fms, ≈ %.1f× faster)\n",
			issue.BaselineMs, issue.CurrentMs, 1.0/m.multiplier))
	}
	return sb.String()
}

func formatFixSection(issue analysis.Issue) string {
	suggestions := suggestionsForPattern(issue.Pattern)
	if len(suggestions) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("🛠 Suggested fixes:\n")
	for _, s := range suggestions {
		sb.WriteString(fmt.Sprintf(" - %s\n", s))
	}
	return sb.String()
}

// suggestionsForPattern maps a pattern identifier to the corresponding templates.
func suggestionsForPattern(pattern string) []string {
	switch pattern {
	case "DATABASE_BOTTLENECK":
		return templates.DBBottleneck()
	case "N_PLUS_ONE_QUERY":
		return templates.N1Query()
	case "EXTERNAL_API_BOTTLENECK":
		return templates.ExternalAPIBottleneck()
	case "PERFORMANCE_REGRESSION":
		return templates.PerformanceRegression()
	case "HIGH_ERROR_RATE":
		return templates.HighErrorRate()
	case "HIGH_LATENCY":
		return templates.HighLatency()
	case "HIGH_INTERNAL_PROCESSING":
		return templates.HighInternalProcessing()
	default:
		return []string{}
	}
}

func formatSummary(result *analysis.Result) string {
	if !result.HasIssues {
		return ""
	}
	critical := 0
	high := 0
	medium := 0
	low := 0
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}
	var parts []string
	if critical > 0 {
		parts = append(parts, fmt.Sprintf("%d critical", critical))
	}
	if high > 0 {
		parts = append(parts, fmt.Sprintf("%d high", high))
	}
	if medium > 0 {
		parts = append(parts, fmt.Sprintf("%d medium", medium))
	}
	if low > 0 {
		parts = append(parts, fmt.Sprintf("%d low", low))
	}
	summary := strings.Join(parts, ", ")
	return fmt.Sprintf("%d issue(s) detected — %s\n", len(result.Issues), summary)
}

func formatFooter(result *analysis.Result) string {
	var sb strings.Builder
	sb.WriteString(strings.Repeat("═", 50) + "\n")
	sb.WriteString(formatSummary(result))
	sb.WriteString(fmt.Sprintf("Analyzed at: %s\n", result.AnalyzedAt.Format("2006-01-02 15:04:05")))
	return sb.String()
}

// truncateSQL truncates a SQL string to maxLen characters, appending "..." if truncated.
func truncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen] + "..."
}
