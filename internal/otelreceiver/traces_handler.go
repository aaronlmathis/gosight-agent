package otelreceiver

import (
	"context"
	"fmt"

	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// HandleTraces processes incoming OTLP traces payloads.
func HandleTraces(ctx context.Context, traces ptrace.Traces) error {
	// Iterate over traces and process them
	traces.ResourceSpans().RemoveIf(func(rs ptrace.ResourceSpans) bool {
		fmt.Printf("Processing resource spans: %v\n", rs)
		return false
	})

	// Forward traces to GoSight server
	conn, err := grpcconn.GetGRPCConn(nil) // Pass the appropriate config
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}
	_ = conn // Placeholder to avoid unused variable error

	fmt.Println("Forwarding traces to GoSight server...")
	return nil
}
