package grpckit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// TestServer provides an in-memory server for testing gRPC and REST endpoints
// without requiring actual network ports.
//
// Example usage:
//
//	func TestMyService(t *testing.T) {
//	    ts, err := grpckit.NewTestServer(
//	        grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
//	            pb.RegisterMyServiceServer(s, &MyService{})
//	        }),
//	        grpckit.WithRESTService(pb.RegisterMyServiceHandlerFromEndpoint),
//	    )
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer ts.Close()
//
//	    // Test gRPC
//	    conn := ts.GRPCClientConn(context.Background())
//	    client := pb.NewMyServiceClient(conn)
//	    resp, err := client.MyMethod(ctx, &pb.MyRequest{})
//
//	    // Test REST
//	    resp, err := ts.HTTPClient().Get(ts.BaseURL() + "/api/v1/resource")
//	}
type TestServer struct {
	*Server
	grpcListener *bufconn.Listener
	httpServer   *httptest.Server
	grpcConn     *grpc.ClientConn
	mu           sync.Mutex
	closed       bool
}

// NewTestServer creates a test server with in-memory connections.
// It accepts the same options as New() but ignores port settings.
func NewTestServer(opts ...Option) (*TestServer, error) {
	// Create the underlying server
	server, err := New(opts...)
	if err != nil {
		return nil, err
	}

	// Create in-memory gRPC listener
	grpcListener := bufconn.Listen(bufSize)

	// Start gRPC server in background
	go func() {
		_ = server.grpcServer.Serve(grpcListener)
	}()

	// Build HTTP handler (similar to startHTTP but without starting a real server)
	httpHandler, err := buildHTTPHandler(server, grpcListener)
	if err != nil {
		server.grpcServer.Stop()
		return nil, err
	}

	// Create httptest server
	httpServer := httptest.NewServer(httpHandler)

	return &TestServer{
		Server:       server,
		grpcListener: grpcListener,
		httpServer:   httpServer,
	}, nil
}

// buildHTTPHandler creates the HTTP handler for the test server.
func buildHTTPHandler(s *Server, grpcListener *bufconn.Listener) (http.Handler, error) {
	ctx := context.Background()

	// Create grpc-gateway mux with marshaler options
	gwMux := runtime.NewServeMux(buildMarshalerOptions(s.cfg)...)

	// Create a dialer that uses the bufconn listener
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return grpcListener.Dial()
	}

	// Register REST services via grpc-gateway using bufconn
	opts := []grpc.DialOption{
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	for _, registrar := range s.cfg.restServices {
		if err := registrar(ctx, gwMux, "bufnet", opts); err != nil {
			return nil, fmt.Errorf("failed to register REST service: %w", err)
		}
	}

	// Create main HTTP mux
	mux := http.NewServeMux()

	// Register health endpoints
	if s.cfg.healthEnabled {
		registerHealthEndpoints(mux, s.healthHandler)
	}

	// Register metrics endpoint
	if s.cfg.metricsEnabled {
		registerMetricsEndpoint(mux)
	}

	// Register custom HTTP handlers
	for _, h := range s.cfg.httpHandlers {
		mux.Handle(h.pattern, h.handler)
	}

	// Mount grpc-gateway mux for all other paths
	mux.Handle("/", gwMux)

	// Build middleware chain
	var handler http.Handler = mux

	// Apply custom HTTP middlewares
	for i := len(s.cfg.httpMiddlewares) - 1; i >= 0; i-- {
		handler = s.cfg.httpMiddlewares[i](handler)
	}

	// Apply built-in auth middleware
	if s.cfg.authFunc != nil {
		handler = authMiddleware(s.cfg, handler)
	}

	// Apply built-in metrics middleware
	if s.cfg.metricsEnabled && s.metrics != nil {
		handler = metricsMiddleware(s.metrics, handler)
	}

	// Apply built-in CORS middleware
	if s.cfg.corsEnabled && s.cfg.corsConfig != nil {
		handler = corsMiddleware(*s.cfg.corsConfig)(handler)
	}

	return handler, nil
}

// GRPCClientConn returns a client connection to the in-memory gRPC server.
// The connection is cached and reused for subsequent calls.
// The caller should NOT close this connection; use TestServer.Close() instead.
func (ts *TestServer) GRPCClientConn(ctx context.Context) *grpc.ClientConn {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.grpcConn != nil {
		return ts.grpcConn
	}

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return ts.grpcListener.Dial()
	}

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		// This should not happen with bufconn, but handle it gracefully
		panic(fmt.Sprintf("failed to dial bufnet: %v", err))
	}

	ts.grpcConn = conn
	return conn
}

// HTTPClient returns an HTTP client configured for the test server.
func (ts *TestServer) HTTPClient() *http.Client {
	return ts.httpServer.Client()
}

// BaseURL returns the base URL for REST requests to the test server.
func (ts *TestServer) BaseURL() string {
	return ts.httpServer.URL
}

// URL constructs a full URL for the given path.
// Example: ts.URL("/api/v1/items") returns "http://127.0.0.1:xxxxx/api/v1/items"
func (ts *TestServer) URL(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return ts.httpServer.URL + path
}

// Close shuts down the test server and releases all resources.
func (ts *TestServer) Close() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return
	}
	ts.closed = true

	// Close gRPC connection if created
	if ts.grpcConn != nil {
		ts.grpcConn.Close()
	}

	// Stop HTTP test server
	ts.httpServer.Close()

	// Stop gRPC server
	ts.grpcServer.Stop()

	// Close the listener
	ts.grpcListener.Close()
}

// MockAuthFunc returns an auth function that accepts specific tokens.
// Use this to easily configure authentication in tests.
//
// Example:
//
//	ts, _ := grpckit.NewTestServer(
//	    grpckit.WithAuth(grpckit.MockAuthFunc("valid-token", "user-123")),
//	    // ... other options
//	)
func MockAuthFunc(validToken, userID string) AuthFunc {
	return func(ctx context.Context, token string) (context.Context, error) {
		if token != validToken {
			return nil, ErrUnauthorized
		}
		return context.WithValue(ctx, "user_id", userID), nil
	}
}

// MockAuthFuncMultiple returns an auth function that accepts multiple tokens.
// Each token maps to a user ID.
//
// Example:
//
//	ts, _ := grpckit.NewTestServer(
//	    grpckit.WithAuth(grpckit.MockAuthFuncMultiple(map[string]string{
//	        "admin-token": "admin-user",
//	        "user-token":  "regular-user",
//	    })),
//	    // ... other options
//	)
func MockAuthFuncMultiple(tokenToUserID map[string]string) AuthFunc {
	return func(ctx context.Context, token string) (context.Context, error) {
		userID, ok := tokenToUserID[token]
		if !ok {
			return nil, ErrUnauthorized
		}
		return context.WithValue(ctx, "user_id", userID), nil
	}
}

// MockAuthFuncAllowAll returns an auth function that accepts any token.
// Useful for tests that don't care about authentication.
func MockAuthFuncAllowAll() AuthFunc {
	return func(ctx context.Context, token string) (context.Context, error) {
		return context.WithValue(ctx, "user_id", "test-user"), nil
	}
}

// TestServerOption is an option specifically for TestServer.
type TestServerOption func(*testServerConfig)

type testServerConfig struct {
	// Future test-specific options can go here
}

// WithTestOption is a placeholder for future test-specific options.
// Currently, TestServer accepts all standard grpckit.Option values.
func WithTestOption() TestServerOption {
	return func(c *testServerConfig) {
		// Placeholder for future test-specific configuration
	}
}
