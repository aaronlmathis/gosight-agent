package otelreceiver

import (
	"context"
	"fmt"

	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"go.opentelemetry.io/collector/pdata/plog"
)

// HandleLogs processes incoming OTLP logs payloads.
func HandleLogs(ctx context.Context, logs plog.Logs) error {
	// Iterate over logs and process them
	logs.ResourceLogs().RemoveIf(func(rl plog.ResourceLogs) bool {
		fmt.Printf("Processing resource logs: %v\n", rl)
		return false
	})

	// Forward logs to GoSight server
	conn, err := grpcconn.GetGRPCConn(nil) // Pass the appropriate config
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}
	_ = conn // Placeholder to avoid unused variable error

	return nil
}
