package otelreceiver

import (
	"context"
	"fmt"
	"net/http"
)

// StartHTTPServer starts the HTTP server for the OTLP receiver.
func StartHTTPServer(ctx context.Context, port int) error {
	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
	}

	http.HandleFunc("/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Handle OTLP metrics payloads
	})

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	return httpServer.ListenAndServe()
}
