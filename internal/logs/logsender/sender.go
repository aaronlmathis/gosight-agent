package logsender

import (
	"context"
	"sync"
	"time"

	"github.com/aaronlmathis/gosight-agent/internal/config"
	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"github.com/aaronlmathis/gosight-agent/internal/otelreceiver"
	"github.com/aaronlmathis/gosight-shared/model"
	"github.com/aaronlmathis/gosight-shared/utils"
	collogpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LogSender holds the gRPC client and connection for OTLP logs.
type LogSender struct {
	client collogpb.LogsServiceClient
	cc     *grpc.ClientConn
	wg     sync.WaitGroup
	cfg    *config.Config
	ctx    context.Context
}

// NewSender initializes a new LogSender and starts the connection manager.
// It returns immediately and launches the background connection manager.
func NewSender(ctx context.Context, cfg *config.Config) (*LogSender, error) {
	s := &LogSender{ctx: ctx, cfg: cfg}
	go s.manageConnection()
	return s, nil
}

// manageConnection dials & maintains the connection, tears it down on global disconnect,
// and retries with exponential backoff up to maxBackoff, then fixed-interval.
func (s *LogSender) manageConnection() {
	const (
		initial    = 1 * time.Second
		maxBackoff = 15 * time.Minute
		factor     = 2
	)

	backoff := initial
	var lastPause time.Time

	for {
		// Check for context cancellation
		select {
		case <-s.ctx.Done():
			utils.Info("Log connection manager shutting down")
			return
		default:
		}

		// If we've just been told to pause (disconnect), tear down connection
		pu := grpcconn.GetPauseUntil()
		if pu.After(lastPause) {
			utils.Info("Global disconnect: closing log connection")
			s.client = nil
			backoff = initial
			lastPause = pu
		}

		// Wait out the pause window (returns when pauseUntil ≤ now)
		grpcconn.WaitForResume()

		// If we already have a client, listen for disconnects
		if s.client != nil {
			select {
			case <-grpcconn.DisconnectNotify():
				utils.Info("Received disconnect signal for logs")
				continue // This will check pause and retry
			case <-s.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				// Periodic check, continue the loop
				continue
			}
		}

		// Try to establish connection
		cc, err := grpcconn.GetGRPCConn(s.cfg)
		if err != nil {
			utils.Info("Server offline (dial): retrying in %s", backoff)

			// Sleep with context cancellation check
			select {
			case <-time.After(backoff):
			case <-s.ctx.Done():
				return
			}

			// Calculate next backoff duration
			if backoff < maxBackoff {
				backoff = time.Duration(float64(backoff) * float64(factor))
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		s.cc = cc
		s.client = collogpb.NewLogsServiceClient(cc)
		utils.Info("OTLP logs client connected")

		// Reset backoff on successful connection
		backoff = initial

		// Brief pause to catch any new disconnects, but allow for context cancellation
		select {
		case <-time.After(time.Second):
		case <-s.ctx.Done():
			return
		}
	}
}

// SendLogs converts the log entries to OTLP format and sends them via unary call.
// If no active client, returns Unavailable so your worker backoff kicks in.
func (s *LogSender) SendLogs(logs []model.LogEntry) error {
	if s.client == nil {
		return status.Error(codes.Unavailable, "no active OTLP logs client")
	}

	// Convert to OTLP format using our conversion function
	otlpReq := otelreceiver.ConvertToOTLPLogs(logs)
	if otlpReq == nil {
		utils.Warn("Failed to convert logs to OTLP format")
		return status.Error(codes.InvalidArgument, "failed to convert logs to OTLP")
	}

	// Send via unary call (OTLP standard)
	utils.Info("Sending %d logs to server via OTLP", len(logs))

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	_, err := s.client.Export(ctx, otlpReq)
	if err != nil {
		utils.Warn("OTLP logs export failed: %v", err)
		return err
	}

	utils.Debug("Successfully exported %d logs via OTLP", len(logs))
	return nil
}

// Close shuts down worker pool and closes the gRPC connection.
func (s *LogSender) Close() error {
	utils.Info("Closing LogSender... waiting for workers")
	s.wg.Wait()
	utils.Info("All LogSender workers finished")
	if s.cc != nil {
		return s.cc.Close()
	}
	return nil
}
