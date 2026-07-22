// analysis/types.go
package analysis

import "time"

// AnalysisInput is the single data model for all rules.
// It is built from multiple queries across traces, queries, and metrics tables.
type AnalysisInput struct {
	ServiceName  string
	Endpoint     string
	Methods      []string
	TotalLatency int64
	DBTime       int64
	ExternalTime int64
	InternalTime int64
	BaselineAvg  float64
	CurrentAvg   float64
	DBQueries    []QueryStat
	ErrorCount   int
	RequestCount int
	ErrorRate    float64
}

// QueryStat represents an aggregated database query for analysis.
// It is intentionally separate from sdk.DBQuery to keep the analysis
// package independent from the SDK.
type QueryStat struct {
	SQL   string
	Count int
	Time  int64
}

// Issue represents a single detected performance issue.
// Output layer (Day 22) formats this struct exactly as defined here.
type Issue struct {
	Pattern    string
	Severity   string
	Confidence string
	Evidence   []string
	Suggestion []string
	BaselineMs float64 // Only set by regression rule
	CurrentMs  float64 // Only set by regression rule
}

// Result is the enriched output of analysis, wrapping issues with context.
// Output layer (Day 22) formats this struct for display.
type Result struct {
	ServiceName  string
	Endpoint     string
	Methods      []string
	AnalyzedAt   time.Time
	Latency      int64
	DBTime       int64
	InternalTime int64
	BaselineAvg  float64
	CurrentAvg   float64
	Issues       []Issue
	HasIssues    bool
}
// EndpointKey identifies a unique (service, endpoint) pair. Analysis and
// storage are scoped by this pair, not endpoint alone, so that two different
// projects using the same endpoint path never share data.
type EndpointKey struct {
	ServiceName string
	Endpoint    string
}