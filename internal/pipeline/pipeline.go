package pipeline

import (
	"context"
	"sync"
	"time"
)

// TelemetryType represents the type of telemetry data.
type TelemetryType string

const (
	Metrics TelemetryType = "metrics"
	Logs    TelemetryType = "logs"
	Traces  TelemetryType = "traces"
)

// TelemetryItem represents a single telemetry item.
type TelemetryItem struct {
	Type TelemetryType
	Data interface{}
}

// Pipeline manages the processing of telemetry data.
type Pipeline struct {
	queue      chan TelemetryItem
	batchSize  int
	batchDelay time.Duration
	mutex      sync.Mutex
}

// NewPipeline creates a new telemetry pipeline.
func NewPipeline(queueSize, batchSize int, batchDelay time.Duration) *Pipeline {
	return &Pipeline{
		queue:      make(chan TelemetryItem, queueSize),
		batchSize:  batchSize,
		batchDelay: batchDelay,
	}
}

// Enqueue adds a telemetry item to the pipeline.
func (p *Pipeline) Enqueue(item TelemetryItem) {
	p.queue <- item
}

// Start begins processing telemetry data.
func (p *Pipeline) Start(ctx context.Context, processFunc func([]TelemetryItem)) {
	go func() {
		batch := make([]TelemetryItem, 0, p.batchSize)
		ticker := time.NewTicker(p.batchDelay)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case item := <-p.queue:
				batch = append(batch, item)
				if len(batch) >= p.batchSize {
					processFunc(batch)
					batch = batch[:0]
				}
			case <-ticker.C:
				if len(batch) > 0 {
					processFunc(batch)
					batch = batch[:0]
				}
			}
		}
	}()
}
