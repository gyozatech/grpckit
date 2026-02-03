package harness

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// VanillaHarness wraps standard grpc + grpc-gateway for benchmarking.
type VanillaHarness struct {
	grpcServer   *grpc.Server
	grpcListener *bufconn.Listener
	httpServer   *httptest.Server
	grpcConn     *grpc.ClientConn
	client       benchpb.BenchmarkServiceClient
}

// VanillaConfig contains configuration options for the vanilla harness.
type VanillaConfig struct {
	// ServiceImpl is the service implementation to use.
	ServiceImpl benchpb.BenchmarkServiceServer
}

// NewVanillaHarness creates a new vanilla grpc + grpc-gateway harness.
func NewVanillaHarness(cfg VanillaConfig) (*VanillaHarness, error) {
	if cfg.ServiceImpl == nil {
		return nil, ErrNilServiceImpl
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	benchpb.RegisterBenchmarkServiceServer(grpcServer, cfg.ServiceImpl)

	// Create in-memory listener
	grpcListener := bufconn.Listen(bufSize)

	// Start gRPC server
	go func() {
		_ = grpcServer.Serve(grpcListener)
	}()

	// Create bufconn dialer
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return grpcListener.Dial()
	}

	// Create gRPC-gateway mux
	ctx := context.Background()
	gwMux := runtime.NewServeMux()

	dialOpts := []grpc.DialOption{
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if err := benchpb.RegisterBenchmarkServiceHandlerFromEndpoint(ctx, gwMux, "bufnet", dialOpts); err != nil {
		grpcServer.Stop()
		return nil, err
	}

	// Create HTTP test server
	httpServer := httptest.NewServer(gwMux)

	// Create gRPC client connection
	grpcConn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		httpServer.Close()
		grpcServer.Stop()
		return nil, err
	}

	client := benchpb.NewBenchmarkServiceClient(grpcConn)

	return &VanillaHarness{
		grpcServer:   grpcServer,
		grpcListener: grpcListener,
		httpServer:   httpServer,
		grpcConn:     grpcConn,
		client:       client,
	}, nil
}

// GRPCClient returns the gRPC client for the benchmark service.
func (h *VanillaHarness) GRPCClient() benchpb.BenchmarkServiceClient {
	return h.client
}

// HTTPClient returns the HTTP client configured for the test server.
func (h *VanillaHarness) HTTPClient() *http.Client {
	return h.httpServer.Client()
}

// BaseURL returns the base URL for REST requests.
func (h *VanillaHarness) BaseURL() string {
	return h.httpServer.URL
}

// Close shuts down the harness and releases resources.
func (h *VanillaHarness) Close() {
	if h.grpcConn != nil {
		h.grpcConn.Close()
	}
	if h.httpServer != nil {
		h.httpServer.Close()
	}
	if h.grpcServer != nil {
		h.grpcServer.Stop()
	}
	if h.grpcListener != nil {
		h.grpcListener.Close()
	}
}
