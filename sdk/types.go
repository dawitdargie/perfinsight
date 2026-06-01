package sdk

type Trace struct {
	TraceID      string
	Endpoint     string
	Latency      int64
	StatusCode   int
	DBTime       int64
	InternalTime int64
}