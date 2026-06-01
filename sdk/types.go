package sdk

import "sync"

type Trace struct {
	TraceID      string
	Endpoint     string
	Latency      int64
	StatusCode   int
	DBTime       int64
	InternalTime int64
}

var (
	traces []Trace
	mu     sync.RWMutex
)

func AddTrace(t Trace) {
	mu.Lock()
	defer mu.Unlock()
	traces = append(traces, t)
}

func GetTraces() []Trace {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Trace, len(traces))
	copy(result, traces)
	return result
}

func ResetTraces() {
	mu.Lock()
	defer mu.Unlock()
	traces = []Trace{}
}