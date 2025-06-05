package otelreceiver

import (
	"context"
	"fmt"

	grpcconn "github.com/aaronlmathis/gosight-agent/internal/grpc"
	"github.com/aaronlmathis/gosight-agent/internal/otelconvert"
	colmetricpb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"github.com/aaronlmathis/gosight-shared/model"
)

// HandleMetrics processes incoming OTLP metrics payloads.
func HandleMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	// Convert pmetric.Metrics to *model.MetricPayload
	payload := &model.MetricPayload{
		// Populate fields from metrics as needed
	}

	// Convert metrics to OTLP format
	otlpReq := otelconvert.ConvertToOTLPMetrics(payload)
	if otlpReq == nil {
		return fmt.Errorf("failed to convert metrics to OTLP format")
	}

	// Forward metrics to GoSight server
	conn, err := grpcconn.GetGRPCConn(nil) // Pass the appropriate config
	if err != nil {
		return fmt.Errorf("failed to get gRPC connection: %w", err)
	}

	// Use the gRPC connection to send metrics to the GoSight server
	client := colmetricpb.NewMetricsServiceClient(conn)
	if _, err := client.Export(ctx, otlpReq); err != nil {
		return fmt.Errorf("failed to send metrics to GoSight server: %w", err)
	}

	return nil
}
