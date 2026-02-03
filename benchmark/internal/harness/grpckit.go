// Package harness provides test harnesses for benchmarking different frameworks.
package harness

import (
	"context"
	"net/http"

	"github.com/gyozatech/grpckit"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"google.golang.org/grpc"
)

// GRPCkitHarness wraps gRPCkit's TestServer for benchmarking.
type GRPCkitHarness struct {
	ts     *grpckit.TestServer
	client benchpb.BenchmarkServiceClient
}

// GRPCkitConfig contains configuration options for the gRPCkit harness.
type GRPCkitConfig struct {
	// EnableAuth enables authentication middleware.
	EnableAuth bool
	// EnableMetrics enables Prometheus metrics middleware.
	EnableMetrics bool
	// EnableCORS enables CORS middleware.
	EnableCORS bool
	// EnableHealth enables health check endpoints.
	EnableHealth bool
	// ServiceImpl is the service implementation to use.
	ServiceImpl benchpb.BenchmarkServiceServer
}

// NewGRPCkitHarness creates a new gRPCkit harness with the given configuration.
func NewGRPCkitHarness(cfg GRPCkitConfig) (*GRPCkitHarness, error) {
	if cfg.ServiceImpl == nil {
		return nil, ErrNilServiceImpl
	}

	opts := []grpckit.Option{
		grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
			benchpb.RegisterBenchmarkServiceServer(s, cfg.ServiceImpl)
		}),
		grpckit.WithRESTService(benchpb.RegisterBenchmarkServiceHandlerFromEndpoint),
	}

	if cfg.EnableAuth {
		opts = append(opts,
			grpckit.WithAuth(grpckit.MockAuthFuncAllowAll()),
			grpckit.WithProtectedEndpoints("/api/**"),
		)
	}

	if cfg.EnableMetrics {
		opts = append(opts, grpckit.WithMetrics())
	}

	if cfg.EnableCORS {
		opts = append(opts, grpckit.WithCORSConfig(grpckit.CORSConfig{
			AllowedOrigins: []string{"*"},
		}))
	}

	if cfg.EnableHealth {
		opts = append(opts, grpckit.WithHealthCheck())
	}

	ts, err := grpckit.NewTestServer(opts...)
	if err != nil {
		return nil, err
	}

	conn := ts.GRPCClientConn(context.Background())
	client := benchpb.NewBenchmarkServiceClient(conn)

	return &GRPCkitHarness{
		ts:     ts,
		client: client,
	}, nil
}

// GRPCClient returns the gRPC client for the benchmark service.
func (h *GRPCkitHarness) GRPCClient() benchpb.BenchmarkServiceClient {
	return h.client
}

// HTTPClient returns the HTTP client configured for the test server.
func (h *GRPCkitHarness) HTTPClient() *http.Client {
	return h.ts.HTTPClient()
}

// BaseURL returns the base URL for REST requests.
func (h *GRPCkitHarness) BaseURL() string {
	return h.ts.BaseURL()
}

// Close shuts down the harness and releases resources.
func (h *GRPCkitHarness) Close() {
	if h.ts != nil {
		h.ts.Close()
	}
}
