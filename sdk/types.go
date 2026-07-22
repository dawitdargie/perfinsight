package sdk

import (
	"sync"
	"time"
)

type Trace struct {
	TraceID      string
	Endpoint     string
	Method       string
	Latency      int64
	StatusCode   int
	DBTime       int64
	ExternalTime int64
	DBQueries    []DBQuery
	InternalTime int64
	ServiceName  string
	Timestamp    time.Time
}

type DBQuery struct {
	SQL   string
	Count int
	Time  int64
}

var (
	activeTraces    = make(map[string]*Trace)
	completedTraces []Trace
	mu              sync.RWMutex
)

// AddTrace registers a new in-flight trace, keyed by its TraceID.
func AddTrace(t Trace) {
	mu.Lock()
	defer mu.Unlock()
	activeTraces[t.TraceID] = &t
}

// GetActiveTrace returns the in-flight trace for the given trace ID, or nil
// if it's not currently tracked (already finalized/removed).
func GetActiveTrace(traceID string) *Trace {
	mu.RLock()
	defer mu.RUnlock()
	return activeTraces[traceID]
}

// RemoveTrace removes an in-flight trace once it's completed.
func RemoveTrace(traceID string) {
	mu.Lock()
	defer mu.Unlock()
	delete(activeTraces, traceID)
}

// recordDBQuery attributes a DB query to the trace with the given ID, if it's
// still active.
func recordDBQuery(traceID, sqlText string, elapsed int64) {
	if traceID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	trace, ok := activeTraces[traceID]
	if !ok {
		return
	}
	for i := range trace.DBQueries {
		if trace.DBQueries[i].SQL == sqlText {
			trace.DBQueries[i].Count++
			trace.DBQueries[i].Time += elapsed
			trace.DBTime += elapsed
			return
		}
	}
	trace.DBQueries = append(trace.DBQueries, DBQuery{SQL: sqlText, Count: 1, Time: elapsed})
	trace.DBTime += elapsed
}

// addCompletedTrace appends a finalized trace (passed by value, already
// fully populated) to an in-memory log for local inspection/testing.
// This is safe (unlike the old design) because it never hands out a pointer
// into shared, still-mutating state — each entry is an immutable snapshot
// appended exactly once, after FinalizeTrace succeeds.
func addCompletedTrace(t Trace) {
	mu.Lock()
	defer mu.Unlock()
	completedTraces = append(completedTraces, t)
}

// GetTraces returns a copy of all completed traces recorded since the last
// ResetTraces call. Intended for local debugging/testing — in production,
// completed traces are shipped to the collector via the exporter, and this
// in-memory log will grow unbounded if never reset, so don't rely on it
// for long-running production processes.
func GetTraces() []Trace {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Trace, len(completedTraces))
	copy(result, completedTraces)
	return result
}

// ActiveTraceCount is useful for diagnostics/tests, e.g. verifying
// in-flight traces don't leak (should trend to 0 under idle load).
func ActiveTraceCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(activeTraces)
}

// ResetTraces clears all in-flight and completed traces. Intended for tests.
func ResetTraces() {
	mu.Lock()
	defer mu.Unlock()
	activeTraces = make(map[string]*Trace)
	completedTraces = nil
}