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

package tracesender

import (
	"context"
	"sync"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
	coltracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
)

const (
	pauseDuration = 1 * time.Minute
	totalCap      = 15 * time.Minute
)

// TraceSender handles OTLP traces and manages gRPC connections.
type TraceSender struct {
	traceClient coltracepb.TraceServiceClient
	cc          *grpc.ClientConn
	wg          sync.WaitGroup
	cfg         *config.Config
	ctx         context.Context
}

// NewSender initializes a new TraceSender and starts a connection manager.
func NewSender(ctx context.Context, cfg *config.Config) (*TraceSender, error) {
	s := &TraceSender{
		ctx: ctx,
		cfg: cfg,
	}
	go s.manageConnection()
	return s, nil
}

// manageConnection handles gRPC connections with backoff.
func (s *TraceSender) manageConnection() {
	const (
		initial    = 1 * time.Second
		maxBackoff = 15 * time.Minute
		factor     = 2
	)

	backoff := initial

	for {
		select {
		case <-s.ctx.Done():
			utils.Info("Trace connection manager shutting down")
			return
		default:
		}

		grpcconn.WaitForResume()

		select {
		case <-grpcconn.DisconnectNotify():
			utils.Info("Global disconnect: closing trace connections")
			if s.cc != nil {
				s.cc.Close()
			}
		}

		conn, err := grpc.Dial(s.cfg.Server.Address, grpc.WithInsecure())
		if err != nil {
			utils.Error("Failed to connect to trace server: %v", err)
			time.Sleep(backoff)
			backoff *= factor
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		s.cc = conn
		s.traceClient = coltracepb.NewTraceServiceClient(conn)
		utils.Info("Connected to trace server")
		backoff = initial
	}
}

// Close shuts down the TraceSender and cleans up resources.
func (s *TraceSender) Close() {
	if s.cc != nil {
		s.cc.Close()
	}
	s.wg.Wait()
}

// StartWorkerPool starts a pool of workers to process trace payloads
func (s *TraceSender) StartWorkerPool(ctx context.Context, taskQueue chan *model.TracePayload, numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case payload := <-taskQueue:
					// Process and send the trace payload
					s.sendTrace(payload)
				}
			}
		}()
	}
}

// sendTrace sends a single trace payload to the server
func (s *TraceSender) sendTrace(payload *model.TracePayload) {
	// Implement the logic to send the trace payload using s.traceClient
	// Placeholder for actual implementation
	utils.Info("Sending trace payload: %v", payload)
}
