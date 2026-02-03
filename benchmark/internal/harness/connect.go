package harness

import (
	"context"
	"net/http"
	"net/http/httptest"

	"connectrpc.com/connect"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"github.com/gyozatech/grpckit/benchmark/proto/gen/benchpbconnect"
)

// ConnectHarness wraps connect-go for benchmarking.
type ConnectHarness struct {
	httpServer    *httptest.Server
	connectClient benchpbconnect.BenchmarkServiceClient
}

// ConnectConfig contains configuration options for the connect harness.
type ConnectConfig struct {
	// ServiceImpl is the service implementation to use.
	ServiceImpl benchpb.BenchmarkServiceServer
}

// connectServiceAdapter adapts the gRPC service interface to connect-go.
type connectServiceAdapter struct {
	impl benchpb.BenchmarkServiceServer
}

// NewConnectHarness creates a new connect-go harness.
func NewConnectHarness(cfg ConnectConfig) (*ConnectHarness, error) {
	if cfg.ServiceImpl == nil {
		return nil, ErrNilServiceImpl
	}

	// Create the adapter
	adapter := &connectServiceAdapter{impl: cfg.ServiceImpl}

	// Create the mux with the connect handler
	mux := http.NewServeMux()
	path, handler := benchpbconnect.NewBenchmarkServiceHandler(adapter)
	mux.Handle(path, handler)

	// Create HTTP test server
	httpServer := httptest.NewServer(mux)

	// Create connect client
	connectClient := benchpbconnect.NewBenchmarkServiceClient(
		httpServer.Client(),
		httpServer.URL,
	)

	return &ConnectHarness{
		httpServer:    httpServer,
		connectClient: connectClient,
	}, nil
}

// ConnectClient returns the connect-go client.
func (h *ConnectHarness) ConnectClient() benchpbconnect.BenchmarkServiceClient {
	return h.connectClient
}

// GRPCClient returns nil as connect-go uses its own client interface.
func (h *ConnectHarness) GRPCClient() benchpb.BenchmarkServiceClient {
	return nil
}

// HTTPClient returns the HTTP client configured for the test server.
func (h *ConnectHarness) HTTPClient() *http.Client {
	return h.httpServer.Client()
}

// BaseURL returns the base URL for requests.
func (h *ConnectHarness) BaseURL() string {
	return h.httpServer.URL
}

// Close shuts down the harness and releases resources.
func (h *ConnectHarness) Close() {
	if h.httpServer != nil {
		h.httpServer.Close()
	}
}

// Implement the connect-go service handler interface.

func (a *connectServiceAdapter) Ping(
	ctx context.Context,
	req *connect.Request[benchpb.PingRequest],
) (*connect.Response[benchpb.PingResponse], error) {
	resp, err := a.impl.Ping(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (a *connectServiceAdapter) Echo(
	ctx context.Context,
	req *connect.Request[benchpb.EchoRequest],
) (*connect.Response[benchpb.EchoResponse], error) {
	resp, err := a.impl.Echo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (a *connectServiceAdapter) GetItem(
	ctx context.Context,
	req *connect.Request[benchpb.GetItemRequest],
) (*connect.Response[benchpb.GetItemResponse], error) {
	resp, err := a.impl.GetItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (a *connectServiceAdapter) CreateItem(
	ctx context.Context,
	req *connect.Request[benchpb.CreateItemRequest],
) (*connect.Response[benchpb.CreateItemResponse], error) {
	resp, err := a.impl.CreateItem(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (a *connectServiceAdapter) ListItems(
	ctx context.Context,
	req *connect.Request[benchpb.ListItemsRequest],
) (*connect.Response[benchpb.ListItemsResponse], error) {
	resp, err := a.impl.ListItems(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
