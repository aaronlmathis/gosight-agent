/*
SPDX-License-Identifier: GPL-3.0-or-later

Copyright (C) 2025 Aaron Mathis

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

package tracerunner

import (
	"context"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	"github.com/aaronlmathis/gosight-agent/internal/traces/tracesender"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
)

// TraceRunner manages the collection and sending of trace data.
type TraceRunner struct {
	Config       *config.Config
	TraceSender  *tracesender.TraceSender
	StartTime    time.Time
	TaskQueue    chan *model.TracePayload
}

// NewRunner initializes a new TraceRunner.
func NewRunner(ctx context.Context, cfg *config.Config) (*TraceRunner, error) {
	traceSender, err := tracesender.NewSender(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &TraceRunner{
		Config:      cfg,
		TraceSender: traceSender,
		StartTime:   time.Now(),
		TaskQueue:   make(chan *model.TracePayload, 500),
	}, nil
}

// Close shuts down the TraceRunner and its sender.
func (r *TraceRunner) Close() {
	if r.TraceSender != nil {
		r.TraceSender.Close()
	}
}

// Enqueue adds a trace payload to the task queue.
func (r *TraceRunner) Enqueue(payload *model.TracePayload) {
	r.TaskQueue <- payload
}

// Run starts the trace collection and sending loop.
func (r *TraceRunner) Run(ctx context.Context) {
	defer r.TraceSender.Close()

	go r.TraceSender.StartWorkerPool(ctx, r.TaskQueue, r.Config.Agent.TraceCollection.Workers)

	ticker := time.NewTicker(r.Config.Agent.TraceCollection.Interval)
	defer ticker.Stop()

	utils.Info("TraceRunner started. Sending traces every %v", r.Config.Agent.TraceCollection.Interval)

	for {
		select {
		case <-ctx.Done():
			utils.Info("TraceRunner shutting down")
			return
		case <-ticker.C:
			// Collect and enqueue trace data here
			utils.Info("Collecting and enqueuing trace data")
		}
	}
}
