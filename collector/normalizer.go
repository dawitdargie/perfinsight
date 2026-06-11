package collector

import (
	"time"

	"github.com/dawitdargie/perfinsight/sdk"
)

func Normalize(t *sdk.Trace) {
	if t.ServiceName == "" {
		t.ServiceName = "unknown"
	}

	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}

	if t.ExternalTime < 0 {
		t.ExternalTime = 0
	}

	if t.DBTime < 0 {
		t.DBTime = 0
	}

	t.InternalTime = t.Latency - t.DBTime - t.ExternalTime
	if t.InternalTime < 0 {
		t.InternalTime = 0
	}

	if t.DBQueries == nil {
		t.DBQueries = []sdk.DBQuery{}
	}
}

func NormalizeBatch(traces []sdk.Trace) {
	for i := range traces {
		Normalize(&traces[i])
	}
}