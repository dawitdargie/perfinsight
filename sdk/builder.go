//sdk/builder.go
package sdk

import (
	"fmt"
	"time"
)

func FinalizeTrace(t *Trace) error {
	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}

	t.InternalTime = t.Latency - t.DBTime - t.ExternalTime
	if t.InternalTime < 0 {
		t.InternalTime = 0
	}

	if t.DBTime > t.Latency {
		return fmt.Errorf("trace %s: DBTime (%dms) exceeds Latency (%dms)", t.TraceID, t.DBTime, t.Latency)
	}

	return nil
}

func SetServiceName(t *Trace, name string) {
	t.ServiceName = name
}