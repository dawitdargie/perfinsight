package collector

import (
	"log"
	"time"
)

type Aggregator struct {
	storage  *Storage
	interval time.Duration
	stopCh   chan struct{}
}

func NewAggregator(storage *Storage, interval time.Duration) *Aggregator {
	return &Aggregator{
		storage:  storage,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (a *Aggregator) Start() {
	go a.run()
}

func (a *Aggregator) run() {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.recalculateBaselines()
		case <-a.stopCh:
			return
		}
	}
}

func (a *Aggregator) recalculateBaselines() {
	keys, err := a.storage.GetEndpoints()
	if err != nil {
		log.Printf("[AGGREGATOR] failed to get endpoints: %v", err)
		return
	}
	for _, key := range keys {
		hourlyAvg, err := a.storage.GetHourlyAverage(key)
		if err != nil {
			log.Printf("[AGGREGATOR] failed to get hourly avg for %s [%s]: %v", key.Endpoint, key.ServiceName, err)
			continue
		}
		if err := a.storage.UpdateBaseline(key, hourlyAvg); err != nil {
			log.Printf("[AGGREGATOR] failed to update baseline for %s [%s]: %v", key.Endpoint, key.ServiceName, err)
			continue
		}
		log.Printf("[AGGREGATOR] updated baseline for %s [%s]: %.2fms", key.Endpoint, key.ServiceName, hourlyAvg)
	}
}

func (a *Aggregator) Stop() {
	close(a.stopCh)
}