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

package metricsender

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/command"
	"github.com/aaronlmathis/gosight-agent/internal/config"
	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"github.com/aaronlmathis/gosight-agent/internal/otelconvert"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/proto"
	"github.com/aaronlmathis/gosight-shared/utils"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	pauseDuration = 1 * time.Minute
	totalCap      = 15 * time.Minute
)

// MetricSender handles OTLP metrics and control commands via dual connections.
type MetricSender struct {
	// OTLP metrics client
	metricsClient colmetricpb.MetricsServiceClient

	// Legacy stream client for commands only
	streamClient proto.StreamServiceClient
	stream       proto.StreamService_StreamClient

	cc  *grpc.ClientConn
	wg  sync.WaitGroup
	cfg *config.Config
	ctx context.Context
}

// NewSender returns immediately and starts a background connection manager.
func NewSender(ctx context.Context, cfg *config.Config) (*MetricSender, error) {
	s := &MetricSender{
		ctx: ctx,
		cfg: cfg,
	}
	go s.manageConnection()
	return s, nil
}

// manageConnection dials/opens connections with backoff, handles global disconnects.
func (s *MetricSender) manageConnection() {
	const (
		initial    = 1 * time.Second
		maxBackoff = 15 * time.Minute
		factor     = 2
	)

	backoff := initial

	for {
		// Check for context cancellation
		select {
		case <-s.ctx.Done():
			utils.Info("Metric connection manager shutting down")
			return
		default:
		}

		// Honor any global pause
		grpcconn.WaitForResume()

		// Handle a global disconnect command
		select {
		case <-grpcconn.DisconnectNotify():
			utils.Info("Global disconnect: closing metric connections")
			if s.stream != nil {
				_ = s.stream.CloseSend()
			}
			s.stream = nil
			s.metricsClient = nil
			backoff = initial
			continue
		default:
		}

		// Ensure we have a live ClientConn
		cc, err := grpcconn.GetGRPCConn(s.cfg)
		if err != nil {
			utils.Info("Server offline (dial): retrying in %s", backoff)

			select {
			case <-time.After(backoff):
			case <-s.ctx.Done():
				return
			}

			if backoff < maxBackoff {
				backoff = time.Duration(float64(backoff) * float64(factor))
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		s.cc = cc

		// Create OTLP metrics client
		s.metricsClient = colmetricpb.NewMetricsServiceClient(cc)

		// Create legacy stream client for commands
		s.streamClient = proto.NewStreamServiceClient(cc)

		// Open the command stream if we don't have one yet
		if s.stream == nil {
			stream, err := s.streamClient.Stream(s.ctx)
			if err != nil {
				utils.Info("Server offline (command stream): retrying in %s", backoff)
				s.metricsClient = nil
				select {
				case <-time.After(backoff):
				case <-s.ctx.Done():
					return
				}

				if backoff < maxBackoff {
					backoff = time.Duration(float64(backoff) * float64(factor))
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
				continue
			}
			s.stream = stream
			utils.Info("Metrics OTLP client and command stream connected")
			backoff = initial
		}

		// Block in the receive loop until error or next disconnect
		s.manageReceive()

		// On exit, close just the stream
		if s.stream != nil {
			_ = s.stream.CloseSend()
		}
		s.stream = nil
		s.metricsClient = nil

		// Log and back off before the next full reconnect
		utils.Info("Metrics connections lost: retrying connect in %s", backoff)

		select {
		case <-time.After(backoff):
		case <-s.ctx.Done():
			return
		}

		if backoff < maxBackoff {
			backoff = time.Duration(float64(backoff) * float64(factor))
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// SendMetrics converts to OTLP and sends via unary call.
func (s *MetricSender) SendMetrics(payload *model.MetricPayload) error {
	if s.metricsClient == nil {
		return status.Error(codes.Unavailable, "no active OTLP metrics client")
	}

	// Convert to OTLP format using our conversion function
	otlpReq := otelconvert.ConvertToOTLPMetrics(payload)
	if otlpReq == nil {
		utils.Warn("Failed to convert metrics to OTLP format")
		return status.Error(codes.InvalidArgument, "failed to convert metrics to OTLP")
	}

	// Send via unary call (OTLP standard)
	utils.Info("Sending %d metrics to server via OTLP", len(payload.Metrics))

	sendCtx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	_, err := s.metricsClient.Export(sendCtx, otlpReq)
	if err != nil {
		utils.Warn("OTLP metrics export failed: %v", err)
		return err
	}

	utils.Debug("Successfully exported %d metrics via OTLP", len(payload.Metrics))
	return nil
}

// manageReceive handles incoming commands; on a disconnect command, broadcasts global pause.
// (COMPLETELY PRESERVED - no changes needed for command handling)
func (s *MetricSender) manageReceive() {
	for {
		resp, err := s.stream.Recv()
		if err != nil {
			if s.ctx.Err() != nil {
				utils.Info("Receive loop canceled")
			} else {
				utils.Error("Stream receive error: %v", err)
			}
			return
		}

		if cmd := resp.Command; cmd != nil &&
			cmd.CommandType == "control" &&
			cmd.Command == "disconnect" {

			utils.Info("Received global disconnect; pausing all senders for %v", pauseDuration)
			grpcconn.PauseConnections(pauseDuration)
			return
		}

		if resp.Command != nil {
			utils.Info("Handling command %s/%s", resp.Command.CommandType, resp.Command.Command)
			if result := command.HandleCommand(s.ctx, resp.Command); result != nil {
				s.sendCommandResponseWithRetry(result)
			}
		}
	}
}

// Close waits for any in-flight work then closes the connection.
func (s *MetricSender) Close() error {
	utils.Info("Closing MetricSender... waiting for workers")
	s.wg.Wait()
	utils.Info("All workers done")
	if s.cc != nil {
		return s.cc.Close()
	}
	return nil
}

// reconnectStream re-dials and reopens the stream for sendCommandResponseWithRetry.
// (PRESERVED - needed for command responses)
func (s *MetricSender) reconnectStream() error {
	if s.cc != nil {
		_ = s.cc.Close()
	}
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	conn, err := grpcconn.GetGRPCConn(s.cfg)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}
	s.cc = conn
	s.streamClient = proto.NewStreamServiceClient(conn)
	s.metricsClient = colmetricpb.NewMetricsServiceClient(conn)

	stream, err := s.streamClient.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to reopen stream: %w", err)
	}
	s.stream = stream
	return nil
}

// sendCommandResponseWithRetry retries CommandResponse up to 3 times with backoff.
// (COMPLETELY PRESERVED - no changes needed)
func (s *MetricSender) sendCommandResponseWithRetry(resp *proto.CommandResponse) {
	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		sendCtx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- s.stream.Send(&proto.StreamPayload{
				Payload: &proto.StreamPayload_CommandResponse{CommandResponse: resp},
			})
		}()

		select {
		case err := <-done:
			if err != nil {
				utils.Warn("CommandResponse send attempt %d failed: %v", attempt, err)
				_ = s.reconnectStream()
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			utils.Debug("CommandResponse sent on attempt %d", attempt)
			return
		case <-sendCtx.Done():
			utils.Warn("CommandResponse send attempt %d timed out", attempt)
			_ = s.reconnectStream()
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}
	utils.Error("Failed to send CommandResponse after %d attempts", maxAttempts)
}
