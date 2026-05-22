package sdk

type Trace struct {
	TraceID      string
	Endpoint     string
	Latency      int64
	DBTime       int64
	InternalTime int64
}