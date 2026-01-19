package grpckit

import (
	"context"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

func TestNew_NoServices(t *testing.T) {
	// Server without any services should fail
	_, err := New()

	if err != ErrServiceNotRegistered {
		t.Errorf("expected ErrServiceNotRegistered, got %v", err)
	}
}

func TestNew_WithGRPCService(t *testing.T) {
	registrarCalled := false
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {
			registrarCalled = true
		}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if !registrarCalled {
		t.Error("expected gRPC service registrar to be called")
	}
}

func TestNew_WithRESTService(t *testing.T) {
	server, err := New(
		WithRESTService(func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			return nil
		}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNew_WithHealthCheck(t *testing.T) {
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.healthHandler == nil {
		t.Error("expected healthHandler to be initialized")
	}

	if !server.healthHandler.IsReady() {
		t.Error("expected server to start ready")
	}
}

func TestNew_WithMetrics(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithMetrics(),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.metrics == nil {
		t.Error("expected metrics to be initialized")
	}
}

func TestNew_WithAuth(t *testing.T) {
	authCalled := false
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithAuth(func(ctx context.Context, token string) (context.Context, error) {
			authCalled = true
			return ctx, nil
		}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.cfg.authFunc == nil {
		t.Error("expected authFunc to be set")
	}

	// Call auth function to verify it was stored correctly
	server.cfg.authFunc(context.Background(), "token")
	if !authCalled {
		t.Error("expected auth function to be callable")
	}
}

func TestServer_SetReady(t *testing.T) {
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Should start ready
	if !server.healthHandler.IsReady() {
		t.Error("expected server to start ready")
	}

	// Set not ready
	server.SetReady(false)
	if server.healthHandler.IsReady() {
		t.Error("expected server to be not ready")
	}

	// Set ready again
	server.SetReady(true)
	if !server.healthHandler.IsReady() {
		t.Error("expected server to be ready again")
	}
}

func TestServer_GRPCServer(t *testing.T) {
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	grpcServer := server.GRPCServer()
	if grpcServer == nil {
		t.Error("expected non-nil gRPC server")
	}
}

func TestServer_HTTPServer_BeforeStart(t *testing.T) {
	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// HTTPServer should be nil before Start is called
	if server.HTTPServer() != nil {
		t.Error("expected nil HTTP server before Start")
	}
}

func TestNew_WithMultipleOptions(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	server, err := New(
		WithGRPCPort(9091),
		WithHTTPPort(8081),
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
		WithMetrics(),
		WithCORS(),
		WithGracefulShutdown(45*time.Second),
		WithLogLevel("debug"),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.cfg.grpcPort != 9091 {
		t.Errorf("expected gRPC port 9091, got %d", server.cfg.grpcPort)
	}

	if server.cfg.httpPort != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", server.cfg.httpPort)
	}

	if !server.cfg.healthEnabled {
		t.Error("expected health to be enabled")
	}

	if !server.cfg.metricsEnabled {
		t.Error("expected metrics to be enabled")
	}

	if !server.cfg.corsEnabled {
		t.Error("expected CORS to be enabled")
	}

	if server.cfg.gracefulTimeout != 45*time.Second {
		t.Errorf("expected 45s timeout, got %v", server.cfg.gracefulTimeout)
	}

	if server.cfg.logLevel != "debug" {
		t.Errorf("expected debug log level, got %s", server.cfg.logLevel)
	}
}

func TestNew_WithInterceptors(t *testing.T) {
	unaryInterceptorCalled := false
	streamInterceptorCalled := false

	server, err := New(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			unaryInterceptorCalled = true
			return handler(ctx, req)
		}),
		WithStreamInterceptor(func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			streamInterceptorCalled = true
			return handler(srv, ss)
		}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if len(server.cfg.unaryInterceptors) != 1 {
		t.Errorf("expected 1 unary interceptor, got %d", len(server.cfg.unaryInterceptors))
	}

	if len(server.cfg.streamInterceptors) != 1 {
		t.Errorf("expected 1 stream interceptor, got %d", len(server.cfg.streamInterceptors))
	}

	// Verify interceptors are stored (not testing execution here)
	_ = unaryInterceptorCalled
	_ = streamInterceptorCalled
}

func TestWrapUnaryInterceptor_NoExceptions(t *testing.T) {
	called := false
	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		called = true
		return handler(ctx, req)
	}

	reg := unaryInterceptorRegistration{
		interceptor:     interceptor,
		exceptEndpoints: nil,
	}

	wrapped := wrapUnaryInterceptor(reg)

	// Should just return the interceptor directly
	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	_, err := wrapped(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected interceptor to be called")
	}
}

func TestWrapUnaryInterceptor_WithExceptions(t *testing.T) {
	interceptorCalled := false
	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		interceptorCalled = true
		return handler(ctx, req)
	}

	reg := unaryInterceptorRegistration{
		interceptor:     interceptor,
		exceptEndpoints: []string{"/test/SkipMethod"},
	}

	wrapped := wrapUnaryInterceptor(reg)

	handler := func(ctx context.Context, req any) (any, error) {
		return "response", nil
	}

	// Call with excepted endpoint - should skip interceptor
	interceptorCalled = false
	_, err := wrapped(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/SkipMethod"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interceptorCalled {
		t.Error("expected interceptor to be skipped for excepted endpoint")
	}

	// Call with non-excepted endpoint - should call interceptor
	interceptorCalled = false
	_, err = wrapped(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/OtherMethod"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !interceptorCalled {
		t.Error("expected interceptor to be called for non-excepted endpoint")
	}
}

func TestWrapStreamInterceptor_NoExceptions(t *testing.T) {
	called := false
	interceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		called = true
		return handler(srv, ss)
	}

	reg := streamInterceptorRegistration{
		interceptor:     interceptor,
		exceptEndpoints: nil,
	}

	wrapped := wrapStreamInterceptor(reg)

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	err := wrapped(nil, nil, &grpc.StreamServerInfo{FullMethod: "/test/Method"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected interceptor to be called")
	}
}

func TestWrapStreamInterceptor_WithExceptions(t *testing.T) {
	interceptorCalled := false
	interceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		interceptorCalled = true
		return handler(srv, ss)
	}

	reg := streamInterceptorRegistration{
		interceptor:     interceptor,
		exceptEndpoints: []string{"/test/SkipStream"},
	}

	wrapped := wrapStreamInterceptor(reg)

	handler := func(srv any, ss grpc.ServerStream) error {
		return nil
	}

	// Call with excepted endpoint - should skip interceptor
	interceptorCalled = false
	err := wrapped(nil, nil, &grpc.StreamServerInfo{FullMethod: "/test/SkipStream"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if interceptorCalled {
		t.Error("expected interceptor to be skipped for excepted endpoint")
	}

	// Call with non-excepted endpoint - should call interceptor
	interceptorCalled = false
	err = wrapped(nil, nil, &grpc.StreamServerInfo{FullMethod: "/test/OtherStream"}, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !interceptorCalled {
		t.Error("expected interceptor to be called for non-excepted endpoint")
	}
}

func TestNew_SamePortMode(t *testing.T) {
	server, err := New(
		WithGRPCPort(8080),
		WithHTTPPort(8080), // Same port = combined mode
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.cfg.grpcPort != server.cfg.httpPort {
		t.Error("expected same port for combined mode")
	}
}

func TestNew_DifferentPortMode(t *testing.T) {
	server, err := New(
		WithGRPCPort(9090),
		WithHTTPPort(8080), // Different ports = separate servers
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)

	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if server.cfg.grpcPort == server.cfg.httpPort {
		t.Error("expected different ports for separate mode")
	}
}
