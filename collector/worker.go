package collector

import (
	"log"
	"sync"

	"github.com/dawitdargie/perfinsight/sdk"
)

type WorkerPool struct {
	traceBuffer chan []sdk.Trace
	workerCount int
	wg          sync.WaitGroup
}

func NewWorkerPool(buffer chan []sdk.Trace, workerCount int) *WorkerPool {
	return &WorkerPool{
		traceBuffer: buffer,
		workerCount: workerCount,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.runWorker(i)
	}
}

func (wp *WorkerPool) runWorker(id int) {
	defer wp.wg.Done()
	for batch := range wp.traceBuffer {
		wp.process(batch)
	}
}

func (wp *WorkerPool) process(batch []sdk.Trace) {
	for _, trace := range batch {
		log.Printf("[WORKER] received trace: id=%s endpoint=%s latency=%dms",
			trace.TraceID, trace.Endpoint, trace.Latency)
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.traceBuffer)
	wp.wg.Wait()
}