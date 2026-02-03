package harness

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// GinHarness wraps Gin + grpc-gateway for benchmarking.
type GinHarness struct {
	grpcServer   *grpc.Server
	grpcListener *bufconn.Listener
	httpServer   *httptest.Server
	grpcConn     *grpc.ClientConn
	client       benchpb.BenchmarkServiceClient
}

// GinConfig contains configuration options for the Gin harness.
type GinConfig struct {
	// ServiceImpl is the service implementation to use.
	ServiceImpl benchpb.BenchmarkServiceServer
}

// NewGinHarness creates a new Gin + grpc-gateway harness.
func NewGinHarness(cfg GinConfig) (*GinHarness, error) {
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

	// Create Gin router
	r := gin.New()

	// Mount grpc-gateway handler under Gin
	r.Any("/api/*path", gin.WrapH(gwMux))

	// Create HTTP test server
	httpServer := httptest.NewServer(r)

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

	return &GinHarness{
		grpcServer:   grpcServer,
		grpcListener: grpcListener,
		httpServer:   httpServer,
		grpcConn:     grpcConn,
		client:       client,
	}, nil
}

// GRPCClient returns the gRPC client for the benchmark service.
func (h *GinHarness) GRPCClient() benchpb.BenchmarkServiceClient {
	return h.client
}

// HTTPClient returns the HTTP client configured for the test server.
func (h *GinHarness) HTTPClient() *http.Client {
	return h.httpServer.Client()
}

// BaseURL returns the base URL for REST requests.
func (h *GinHarness) BaseURL() string {
	return h.httpServer.URL
}

// Close shuts down the harness and releases resources.
func (h *GinHarness) Close() {
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
