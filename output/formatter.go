package output

import (
	"fmt"
	"math"
	"strings"

	"github.com/dawitdargie/perfinsight/analysis"
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
	sb.WriteString(fmt.Sprintf("\n%s %s\n\n", icon, title))

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
		sb.WriteString(fmt.Sprintf(" - %s\n", e))
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

// formatFixSection formats fix suggestions.
// Fully implemented Day 25 using template system.
func formatFixSection(issue analysis.Issue) string {
	if len(issue.Suggestion) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("🛠 Suggestions:\n")
	for _, s := range issue.Suggestion {
		sb.WriteString(fmt.Sprintf(" - %s\n", s))
	}
	return sb.String()
}

func formatFooter(result *analysis.Result) string {
	return fmt.Sprintf("Analyzed at: %s\n", result.AnalyzedAt.Format("2006-01-02 15:04:05"))
}