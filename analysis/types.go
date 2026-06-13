package analysis

// AnalysisInput is the single data model for all rules.
// It is built from multiple queries across traces, queries, and metrics tables.
type AnalysisInput struct {
	Endpoint     string
	TotalLatency int64
	DBTime       int64
	ExternalTime int64
	InternalTime int64
	BaselineAvg  float64
	CurrentAvg   float64
	DBQueries    []QueryStat
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
}
