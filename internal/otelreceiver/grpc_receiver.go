package otelreceiver

// import (
// 	"context"
// 	"fmt"
// 	"net"

// 	"github.com/aaronlmathis/gosight-agent/internal/config"
// 	"github.com/aaronlmathis/gosight-agent/internal/traces/tracerunner"
// 	"github.com/aaronlmathis/gosight-shared/model"
// 	"go.opentelemetry.io/collector/receiver/otlpreceiver"
// 	"google.golang.org/grpc"
// )

// // Define traceRunner as a global variable
// var traceRunner *tracerunner.TraceRunner

// type OtlpGrpcReceiver struct {
// 	// Add any necessary fields here, such as configuration or state
// 	cfg *config.Config  // Configuration for the receiver
// 	ctx context.Context // Context for graceful shutdown
// }

// // StartGRPCServer starts the gRPC server for the OTLP receiver.
// func StartGRPCServer(ctx context.Context, cfg *config.Config) error {
// 	var err error
// 	traceRunner, err = tracerunner.NewRunner(ctx, cfg)
// 	if err != nil {
// 		return fmt.Errorf("failed to initialize Tracerunner: %w", err)
// 	}

// 	defer traceRunner.Close()

// 	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
// 	if err != nil {
// 		return fmt.Errorf("failed to start gRPC listener: %w", err)
// 	}

// 	server := grpc.NewServer()

// 	// Create OTLP receiver factory
// 	receiverFactory := otlpreceiver.NewFactory()

// 	// Register the receiver with the gRPC server
// 	if err := receiverFactory.CreateDefaultConfig(); err != nil {
// 		return fmt.Errorf("failed to create default config for OTLP receiver: %v", err)
// 	}

// 	// Register a handler for trace payloads
// 	server.RegisterService(&grpc.ServiceDesc{
// 		ServiceName: "TraceService",
// 		HandlerType: (*TraceServiceServer)(nil),
// 		Methods: []grpc.MethodDesc{
// 			{
// 				MethodName: "Export",
// 				Handler:    traceExportHandler,
// 			},
// 		},
// 	}, nil)

// 	go func() {
// 		<-ctx.Done()
// 		server.GracefulStop()
// 	}()

// 	return server.Serve(listener)
// }

// // Implement a function matching grpc.MethodHandler type
// func traceExportHandler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
// 	tracePayload := &model.TracePayload{}
// 	if err := dec(tracePayload); err != nil {
// 		return nil, err
// 	}

// 	if traceRunner != nil {
// 		traceRunner.Enqueue(tracePayload)
// 	}
// 	return nil, nil
// }

// // Define the TraceServiceServer interface
// type TraceServiceServer interface {
// 	Export(stream grpc.ServerStream) error
// }
