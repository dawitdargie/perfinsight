package sdk

import (
	"io"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

type Exporter struct {
	collectorURL string
	buffer       chan Trace
	stopCh       chan struct{}
	client       *http.Client
	wg           sync.WaitGroup
}

func NewExporter(collectorURL string) *Exporter {
	e := &Exporter{
		collectorURL: collectorURL,
		buffer:       make(chan Trace, 1000),
		stopCh:       make(chan struct{}),
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	e.wg.Add(1)
	go e.run()

	return e
}

func (e *Exporter) Enqueue(t Trace) {
	select {
	case e.buffer <- t:
	default:
	fmt.Fprintln(os.Stderr, "perfinsight: trace buffer full, dropping trace")	}
}

func (e *Exporter) run() {
	defer e.wg.Done()

	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()

	var batch []Trace

	for {
		select {
		case <-ticker.C:
			if len(batch) > 0 {
				e.send(batch)
				batch = nil
			}

		case trace := <-e.buffer:
			batch = append(batch, trace)
			if len(batch) >= 1 {
				e.send(batch)
				batch = nil
			}

		case <-e.stopCh:
			for {
				select {
				case trace := <-e.buffer:
					batch = append(batch, trace)
				default:
					if len(batch) > 0 {
						e.send(batch)
					}
					return
				}
			}
		}
	}
}

func (e *Exporter) send(traces []Trace) {
	body, err := json.Marshal(traces)
	if err != nil {
		fmt.Fprintf(os.Stderr, "perfinsight: failed to marshal traces: %v\n", err)
		return
	}

	resp, err := e.client.Post(
	e.collectorURL+"/ingest-trace",
	"application/json",
	bytes.NewReader(body),
)
if err != nil {
	fmt.Fprintf(os.Stderr, "perfinsight: failed to send traces: %v\n", err)
	return
}
defer resp.Body.Close()

if resp.StatusCode >= 300 {
	body, _ := io.ReadAll(resp.Body)

	fmt.Fprintf(
		os.Stderr,
		"perfinsight: collector returned status %d: %s\n",
		resp.StatusCode,
		string(body),
	)
}
}

func (e *Exporter) Close() {
	close(e.stopCh)
	e.wg.Wait()
}