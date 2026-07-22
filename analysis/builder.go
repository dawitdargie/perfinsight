// analysis/builder.go
package analysis

import "time"

// BuildResult assembles analysis input and issues into a structured Result.
func BuildResult(input AnalysisInput, issues []Issue) *Result {
	return &Result{
		ServiceName:  input.ServiceName,
		Endpoint:     input.Endpoint,
		AnalyzedAt:   time.Now(),
		Latency:      input.TotalLatency,
		DBTime:       input.DBTime,
		InternalTime: input.InternalTime,
		BaselineAvg:  input.BaselineAvg,
		CurrentAvg:   input.CurrentAvg,
		Issues:       issues,
		HasIssues:    len(issues) > 0,
	}
}