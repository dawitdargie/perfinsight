package output

import (
	"strings"
	"testing"
	"time"

	"github.com/dawitdargie/perfinsight/analysis"
)

func TestFormatResult_NoIssuesReturnsCleanMessage(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/fast",
		HasIssues: false,
	}
	output := FormatResult(result)
	if !strings.Contains(output, "No performance issues") {
		t.Error("Expected clean message for no issues")
	}
	if !strings.Contains(output, "/fast") {
		t.Error("Expected endpoint name in output")
	}
}

func TestFormatResult_NilResultHandledSafely(t *testing.T) {
	output := FormatResult(nil)
	if output == "" {
		t.Error("Expected non-empty output for nil result")
	}
}

func TestFormatResult_HeaderContainsEndpoint(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		Latency:   320,
		DBTime:    280,
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "DATABASE_BOTTLENECK",
				Severity:   "high",
				Evidence:   []string{"DB time: 280ms (87%)"},
				Suggestion: []string{"Add indexes"},
			},
		},
	}
	output := FormatResult(result)
	if !strings.Contains(output, "/orders") {
		t.Error("Expected endpoint in output")
	}
	if !strings.Contains(output, "320") {
		t.Error("Expected latency in output")
	}
}

func TestFormatResult_SeverityIconPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{"DB slow"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "🟠") {
		t.Error("Expected orange icon for high severity")
	}
}

func TestFormatResult_CriticalSeverityShowsRedIcon(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "N_PLUS_ONE_QUERY",
				Severity: "critical",
				Evidence: []string{"Query repeated 150x"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "🔴") {
		t.Error("Expected red icon for critical severity")
	}
}

func TestFormatResult_EvidenceSectionPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{"DB time: 280ms (87% of total)"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Evidence") {
		t.Error("Expected Evidence section")
	}
	if !strings.Contains(out, "DB time: 280ms") {
		t.Error("Expected evidence content")
	}
}

func TestFormatResult_MultipleIssuesSeparated(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{Pattern: "DATABASE_BOTTLENECK", Severity: "high", Evidence: []string{"DB slow"}},
			{Pattern: "N_PLUS_ONE_QUERY", Severity: "critical", Evidence: []string{"Query 150x"}},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Database Bottleneck") {
		t.Error("Expected first issue title")
	}
	if !strings.Contains(out, "N+1 Query Pattern") {
		t.Error("Expected second issue title")
	}
	if !strings.Contains(out, "─") {
		t.Error("Expected separator between issues")
	}
}

func TestFormatResult_RegressionShowsMultiplier(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "PERFORMANCE_REGRESSION",
				Severity:   "critical",
				BaselineMs: 100,
				CurrentMs:  320,
				Evidence:   []string{"3.2x slower"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "3.2×") {
		t.Error("Expected multiplier in regression output")
	}
	if !strings.Contains(out, "slower") {
		t.Error("Expected 'slower' in regression output")
	}
	if !strings.Contains(out, "100ms → 320ms") {
		t.Error("Expected before/after values in output")
	}
}

func TestFormatResult_ImprovementShowsPercentage(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/checkout",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "PERFORMANCE_REGRESSION",
				Severity:   "critical",
				BaselineMs: 200,
				CurrentMs:  140, // Faster — improvement
				Evidence:   []string{"30% faster"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "faster") {
		t.Error("Expected 'faster' in improvement output")
	}
	if !strings.Contains(out, "%") {
		t.Error("Expected percentage in improvement output")
	}
}

func TestFormatResult_NoChangeSectionWithoutBaseline(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:    "DATABASE_BOTTLENECK",
				Severity:   "high",
				BaselineMs: 0, // No baseline
				Evidence:   []string{"DB slow"},
			},
		},
	}
	out := FormatResult(result)
	if strings.Contains(out, "📊 Change") {
		t.Error("Should not show Change section without baseline")
	}
}

func TestFormatResult_EmptyEvidenceSkipsSection(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{},
			},
		},
	}
	out := FormatResult(result)
	if strings.Contains(out, "🔍 Evidence:") {
		t.Error("Should not show Evidence heading with empty evidence")
	}
}

func TestFormatResult_LongSQLTruncated(t *testing.T) {
	longSQL := strings.Repeat("SELECT * FROM very_long_table_name WHERE id = $1 AND ", 5)
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "N_PLUS_ONE_QUERY",
				Severity: "critical",
				Evidence: []string{"SQL: " + longSQL},
			},
		},
	}
	out := FormatResult(result)
	if strings.Contains(out, longSQL) {
		t.Error("Long SQL should be truncated")
	}
	if !strings.Contains(out, "...") {
		t.Error("Truncated SQL should end with ...")
	}
}

func TestFormatResult_PatternExplanationPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "N_PLUS_ONE_QUERY",
				Severity: "critical",
				Evidence: []string{"Query 150x"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "repeatedly in a loop") {
		t.Error("Expected pattern explanation in output")
	}
}

func TestTruncateSQL_TruncatesLongString(t *testing.T) {
	long := strings.Repeat("x", 200)
	result := truncateSQL(long, 80)
	if len(result) > 83 { // 80 + "..."
		t.Errorf("Expected max 83 chars, got %d", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Error("Expected ... suffix")
	}
}

func TestTruncateSQL_LeavesShortStringUnchanged(t *testing.T) {
	short := "SELECT * FROM orders"
	result := truncateSQL(short, 80)
	if result != short {
		t.Errorf("Expected unchanged string, got %s", result)
	}
}

func TestFormatResult_FixSectionPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "DATABASE_BOTTLENECK",
				Severity: "high",
				Evidence: []string{"DB slow"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Suggested fixes") {
		t.Error("Expected fix section in output")
	}
}

func TestFormatResult_N1FixMentionsBatch(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{
				Pattern:  "N_PLUS_ONE_QUERY",
				Severity: "critical",
				Evidence: []string{"Query 150x"},
			},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "batch") {
		t.Error("Expected batch suggestion in N+1 output")
	}
}

func TestFormatResult_SummaryShowsCorrectCounts(t *testing.T) {
	result := &analysis.Result{
		Endpoint:  "/orders",
		HasIssues: true,
		Issues: []analysis.Issue{
			{Pattern: "N_PLUS_ONE_QUERY", Severity: "critical", Evidence: []string{"x"}},
			{Pattern: "DATABASE_BOTTLENECK", Severity: "high", Evidence: []string{"x"}},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "2 issue(s)") {
		t.Error("Expected issue count in summary")
	}
	if !strings.Contains(out, "1 critical") {
		t.Error("Expected critical count")
	}
	if !strings.Contains(out, "1 high") {
		t.Error("Expected high count")
	}
}

func TestFormatResult_TimestampPresent(t *testing.T) {
	result := &analysis.Result{
		Endpoint:   "/orders",
		HasIssues:  true,
		AnalyzedAt: time.Now(),
		Issues: []analysis.Issue{
			{Pattern: "DATABASE_BOTTLENECK", Severity: "high", Evidence: []string{"x"}},
		},
	}
	out := FormatResult(result)
	if !strings.Contains(out, "Analyzed at:") {
		t.Error("Expected timestamp in output")
	}
}
