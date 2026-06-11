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
	storage     *Storage
}

func NewWorkerPool(buffer chan []sdk.Trace, workerCount int, storage *Storage) *WorkerPool {
	return &WorkerPool{
		traceBuffer: buffer,
		workerCount: workerCount,
		storage:     storage,
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
	NormalizeBatch(batch)
	for _, trace := range batch {
		if err := wp.storage.Save(trace); err != nil {
			log.Printf("[WORKER] failed to save trace %s: %v", trace.TraceID, err)
		}
	}
}

func (wp *WorkerPool) Stop() {
	close(wp.traceBuffer)
	wp.wg.Wait()
}