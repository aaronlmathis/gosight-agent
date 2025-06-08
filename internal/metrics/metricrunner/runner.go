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
// agent/internal/metrics/metricrunner/runner.go
package metricrunner

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	"github.com/aaronlmathis/gosight-agent/internal/metrics/metriccollector"
	"github.com/aaronlmathis/gosight-agent/internal/metrics/metricsender"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
)

// MetricRunner is a struct that handles the collection and sending of metrics.
// It manages the metric collection interval, the task queue, and the
// metric sender. It implements the Run method to start the collection process
// and the Close method to clean up resources.
type MetricRunner struct {
	Config         *config.Config
	MetricSender   *metricsender.MetricSender
	MetricRegistry *metriccollector.MetricRegistry
	StartTime      time.Time
	Meta           *model.Meta
}

// NewRunner creates a new MetricRunner instance.
// It initializes the metric sender and sets up the context for the runner.
// It returns a pointer to the MetricRunner and an error if any occurs during initialization.
// The MetricRunner is responsible for collecting and sending metrics to the server.
func NewRunner(ctx context.Context, cfg *config.Config, baseMeta *model.Meta) (*MetricRunner, error) {

	// Init the collector registry
	metricRegistry := metriccollector.NewRegistry(cfg)

	// Init Metric Sender
	metricSender, err := metricsender.NewSender(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender: %v", err)
	}

	return &MetricRunner{
		Config:         cfg,
		MetricSender:   metricSender,
		MetricRegistry: metricRegistry,
		StartTime:      time.Now(),
		Meta:           baseMeta,
	}, nil
}

// Close closes the metric sender.
// It cleans up resources and ensures that the sender is properly closed.
// This is important to prevent resource leaks and ensure that all data is sent before shutting down.
func (r *MetricRunner) Close() {
	if r.MetricSender != nil {
		_ = r.MetricSender.Close()
	}
}

// RunAgent starts the agent's collection loop and sends tasks to the pool of workers.
// It collects metrics at the specified interval and sends them to the server.
// The method runs indefinitely until the context is done.
// It handles the collection of both host and container metrics.
// The host metrics are sent as a single payload, while container metrics are sent as separate payloads.
func (r *MetricRunner) Run(ctx context.Context) {
	defer r.MetricSender.Close()

	// Change queue to handle metric batches instead of payloads
	taskQueue := make(chan []*model.Metric, 500)
	go r.MetricSender.StartWorkerPool(ctx, taskQueue, r.Config.Agent.MetricCollection.Workers)

	ticker := time.NewTicker(r.Config.Agent.MetricCollection.Interval)
	defer ticker.Stop()

	utils.Info("MetricRunner started. Sending metrics every %v", r.Config.Agent.MetricCollection.Interval)

	for {
		select {
		case <-ctx.Done():
			utils.Warn("agent shutting down...")
			return
		case <-ticker.C:
			metrics, err := r.MetricRegistry.Collect(ctx)
			if err != nil {
				utils.Error("metric collection failed: %v", err)
				continue
			}

			var hostMetrics []*model.Metric
			containerBatches := make(map[string][]*model.Metric)

			for _, m := range metrics {
				// Make a copy of the metric to avoid pointer issues
				metricCopy := m

				// Check if this metric has container_id in any of its data points
				var containerID string
				var isContainerMetric bool

				// Look through all data points for container_id
				for _, dp := range metricCopy.DataPoints {
					if dp.Attributes != nil {
						if id, exists := dp.Attributes["container_id"]; exists && id != "" {
							containerID = id
							isContainerMetric = true
							break
						}
					}
				}

				if isContainerMetric {
					// Add container metrics to containerBatches
					containerBatches[containerID] = append(containerBatches[containerID], &metricCopy)
				} else {
					// Host metrics
					hostMetrics = append(hostMetrics, &metricCopy)
				}
			}

			// Send host metrics as a single batch
			if len(hostMetrics) > 0 {
				select {
				case taskQueue <- hostMetrics:
				default:
					utils.Warn("Host task queue full! Dropping host metrics")
				}
			}

			// Send each container as a separate batch
			for containerID, metrics := range containerBatches {
				select {
				case taskQueue <- metrics:
				default:
					utils.Warn("Task queue full! Dropping container metrics for %s", containerID)
				}
			}
		}
	}
}
