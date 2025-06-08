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

// gosight/agent/internal/sender/task.go
//

package metricsender

import (
	"context"
	"time"

	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
)

// StartWorkerPool launches N workers and processes metric batches with retries
// in case of transient errors. Each worker will attempt to send the batch
// to the gRPC server. The number of workers is determined by the workerCount
// parameter. The workers will run until the context is done or an error occurs.
// The function uses a goroutine for each worker, allowing them to run concurrently.
func (s *MetricSender) StartWorkerPool(ctx context.Context, queue <-chan []*model.Metric, workerCount int) {
	for i := 0; i < workerCount; i++ {
		s.wg.Add(1)
		go func(id int) {
			defer s.wg.Done()
			for {
				// Exit if the runner context is done
				select {
				case <-ctx.Done():
					utils.Info("Metric worker #%d shutting down", id)
					return
				default:
				}

				// If not connected, wait and retry
				if s.metricsClient == nil {
					time.Sleep(500 * time.Millisecond)
					continue
				}

				// Pull next batch (or exit)
				var batch []*model.Metric
				select {
				case batch = <-queue:
				case <-ctx.Done():
					utils.Info("Metric worker #%d shutting down", id)
					return
				}

				// Send the batch (errors will be logged)
				if err := s.SendMetrics(batch); err != nil {
					utils.Warn("Metric worker #%d failed to send batch: %v", id, err)
				}
			}
		}(i + 1)
	}
}
