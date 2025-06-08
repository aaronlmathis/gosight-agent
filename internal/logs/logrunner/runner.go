/*
SPDX-License-Identifier: GPL-3.0-or-later

Copyright (C) 2025 Aaron Mathis aaron.mathis@gmail.com

This file is part of GoSight.

GoSight is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

GoSight is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with GoSight. If not, see https://www.gnu.org/licenses/.
*/
package logrunner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	"github.com/aaronlmathis/gosight-agent/internal/logs/logcollector"
	"github.com/aaronlmathis/gosight-agent/internal/logs/logsender"
	"github.com/aaronlmathis/gosight-agent/internal/meta"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
)

// LogRunner is a struct that handles the collection and sending of log data.
// It manages the log collection interval, the task queue, and the
// log sender. It implements the Run method to start the collection process
// and the Close method to clean up resources.
type LogRunner struct {
	Config      *config.Config
	LogSender   *logsender.LogSender
	LogRegistry *logcollector.LogRegistry
	Meta        *model.Meta
	runWg       sync.WaitGroup
}

// NewRunner creates a new LogRunner instance.
// It initializes the log sender and sets up the context for the runner.
// It returns a pointer to the LogRunner and an error if any occurs during initialization.
func NewRunner(ctx context.Context, cfg *config.Config, baseMeta *model.Meta) (*LogRunner, error) {

	logRegistry := logcollector.NewRegistry(cfg)

	logSender, err := logsender.NewSender(ctx, cfg)
	if err != nil {
		// Clean up registry if sender fails?
		logRegistry.Close() // Add a Close method to LogRegistry
		return nil, fmt.Errorf("failed to create sender: %v", err)
	}

	return &LogRunner{
		Config:      cfg,
		LogSender:   logSender,
		LogRegistry: logRegistry,
		Meta:        baseMeta,
	}, nil
}

// Close cleans up the resources used by the LogRunner.
// It closes the log sender and the log registry.
// It should be called when the LogRunner is no longer needed.
func (r *LogRunner) Close() {
	utils.Info("Closing Log Runner...")

	// Close collectors first to stop feeding new logs
	if r.LogRegistry != nil {
		r.LogRegistry.Close() // Ensure LogRegistry has a Close method that calls Close on all collectors
	}

	// Close the sender (which should handle its worker pool shutdown)
	if r.LogSender != nil {
		if err := r.LogSender.Close(); err != nil {
			utils.Error("Error closing log sender: %v", err)
		}
	}

	// Wait for sender pool goroutines to finish (if Close doesn't block)
	// Or manage worker shutdown signalling more explicitly if needed.
	// The LogSender's Close method should ideally handle this wait.
	utils.Info("Log Runner closed.")

}

func (r *LogRunner) Run(ctx context.Context) {
	defer r.LogSender.Close()

	// Change queue to handle log batches instead of payloads
	taskQueue := make(chan []model.LogEntry, 500)
	go r.LogSender.StartWorkerPool(ctx, taskQueue, r.Config.Agent.LogCollection.Workers)

	ticker := time.NewTicker(r.Config.Agent.LogCollection.Interval)
	defer ticker.Stop()

	utils.Info("LogRunner started. Collecting logs every %v", r.Config.Agent.LogCollection.Interval)

	for {
		select {
		case <-ctx.Done():
			utils.Warn("log runner shutting down...")
			return
		case <-ticker.C:
			// Collect logs from all collectors
			logBatches, err := r.LogRegistry.Collect(ctx)
			if err != nil {
				utils.Error("log collection failed: %v", err)
				continue
			}

			// Process each batch and add Meta information
			for _, batch := range logBatches {
				if len(batch) == 0 {
					continue
				}

				// Enrich each log entry with Meta information
				enrichedBatch := make([]model.LogEntry, len(batch))
				for i, logEntry := range batch {
					enrichedBatch[i] = logEntry

					// Set Meta if not already present
					if enrichedBatch[i].Meta == nil {
						enrichedBatch[i].Meta = r.Meta
					} else {
						// Merge with base Meta, preserving log-specific metadata
						enrichedBatch[i].Meta = meta.MergeMetaWithBase(r.Meta, enrichedBatch[i].Meta)
					}
				}

				// Send the batch
				select {
				case taskQueue <- enrichedBatch:
				default:
					utils.Warn("Log task queue full! Dropping log batch with %d entries", len(enrichedBatch))
				}
			}
		}
	}
}
