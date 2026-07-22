// collector/validation.go
package collector

import (
	"fmt"
	"os"

	"github.com/dawitdargie/perfinsight/sdk"
)

type ValidationError struct {
	TraceID string
	Field   string
	Reason  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("trace %s invalid: field=%s reason=%s", e.TraceID, e.Field, e.Reason)
}

func ValidateTrace(t sdk.Trace) error {
	if t.TraceID == "" {
		return &ValidationError{TraceID: "unknown", Field: "TraceID", Reason: "must not be empty"}
	}

	if t.Endpoint == "" {
		return &ValidationError{TraceID: t.TraceID, Field: "Endpoint", Reason: "must not be empty"}
	}

	if t.Endpoint[0] != '/' {
		return &ValidationError{TraceID: t.TraceID, Field: "Endpoint", Reason: "must start with /"}
	}

	if t.Latency <= 0 {
		return &ValidationError{TraceID: t.TraceID, Field: "Latency", Reason: "must be > 0"}
	}

	if t.DBTime > t.Latency {
		return &ValidationError{TraceID: t.TraceID, Field: "DBTime", Reason: fmt.Sprintf("(%d) exceeds Latency (%d)", t.DBTime, t.Latency)}
	}

	if t.ExternalTime > t.Latency {
		return &ValidationError{TraceID: t.TraceID, Field: "ExternalTime", Reason: fmt.Sprintf("(%d) exceeds Latency (%d)", t.ExternalTime, t.Latency)}
	}

	if t.DBTime+t.ExternalTime > t.Latency {
		return &ValidationError{TraceID: t.TraceID, Field: "DBTime+ExternalTime", Reason: "combined exceeds TotalLatency"}
	}

	if t.StatusCode < 100 || t.StatusCode > 599 {
		return &ValidationError{TraceID: t.TraceID, Field: "StatusCode", Reason: fmt.Sprintf("(%d) is not a valid HTTP status", t.StatusCode)}
	}

	return nil
}

func ValidateBatch(traces []sdk.Trace) []sdk.Trace {
	valid := make([]sdk.Trace, 0, len(traces))
	for _, t := range traces {
		if err := ValidateTrace(t); err != nil {
			fmt.Fprintf(os.Stderr, "perfinsight: %v\n", err)
			continue
		}
		valid = append(valid, t)
	}
	return valid
}