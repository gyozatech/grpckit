package grpckit

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func TestNewServerConfig(t *testing.T) {
	cfg := newServerConfig()

	if cfg.grpcPort != 9090 {
		t.Errorf("expected default gRPC port 9090, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != 8080 {
		t.Errorf("expected default HTTP port 8080, got %d", cfg.httpPort)
	}
	if cfg.gracefulTimeout != 30*time.Second {
		t.Errorf("expected default graceful timeout 30s, got %v", cfg.gracefulTimeout)
	}
	if cfg.logLevel != "info" {
		t.Errorf("expected default log level info, got %s", cfg.logLevel)
	}
	if cfg.grpcServices == nil {
		t.Error("expected grpcServices to be initialized")
	}
	if cfg.restServices == nil {
		t.Error("expected restServices to be initialized")
	}
	if cfg.marshalers == nil {
		t.Error("expected marshalers to be initialized")
	}
}

func TestWithGRPCPort(t *testing.T) {
	cfg := newServerConfig()
	opt := WithGRPCPort(9091)
	opt(cfg)

	if cfg.grpcPort != 9091 {
		t.Errorf("expected gRPC port 9091, got %d", cfg.grpcPort)
	}
}

func TestWithHTTPPort(t *testing.T) {
	cfg := newServerConfig()
	opt := WithHTTPPort(8081)
	opt(cfg)

	if cfg.httpPort != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", cfg.httpPort)
	}
}

func TestWithGRPCService(t *testing.T) {
	cfg := newServerConfig()

	registrarCalled := false
	opt := WithGRPCService(func(s grpc.ServiceRegistrar) {
		registrarCalled = true
	})
	opt(cfg)

	if len(cfg.grpcServices) != 1 {
		t.Errorf("expected 1 gRPC service, got %d", len(cfg.grpcServices))
	}

	// Call the registrar to verify it was stored correctly
	cfg.grpcServices[0].registrar(nil)
	if !registrarCalled {
		t.Error("expected registrar to be called")
	}
}

func TestWithRESTService(t *testing.T) {
	cfg := newServerConfig()

	registrarCalled := false
	opt := WithRESTService(func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
		registrarCalled = true
		return nil
	})
	opt(cfg)

	if len(cfg.restServices) != 1 {
		t.Errorf("expected 1 REST service, got %d", len(cfg.restServices))
	}

	// Call the registrar to verify
	_ = cfg.restServices[0](context.Background(), nil, "", nil)
	if !registrarCalled {
		t.Error("expected REST registrar to be called")
	}
}

func TestWithAuth(t *testing.T) {
	cfg := newServerConfig()

	authCalled := false
	opt := WithAuth(func(ctx context.Context, token string) (context.Context, error) {
		authCalled = true
		return ctx, nil
	})
	opt(cfg)

	if cfg.authFunc == nil {
		t.Error("expected authFunc to be set")
	}

	_, _ = cfg.authFunc(context.Background(), "token")
	if !authCalled {
		t.Error("expected auth function to be called")
	}
}

func TestWithProtectedEndpoints(t *testing.T) {
	cfg := newServerConfig()

	opt := WithProtectedEndpoints("/api/v1/*", "/admin/**")
	opt(cfg)

	if len(cfg.protectedEndpoints) != 2 {
		t.Errorf("expected 2 protected endpoints, got %d", len(cfg.protectedEndpoints))
	}

	if cfg.protectedEndpoints[0] != "/api/v1/*" {
		t.Errorf("expected first endpoint /api/v1/*, got %s", cfg.protectedEndpoints[0])
	}
}

func TestWithPublicEndpoints(t *testing.T) {
	cfg := newServerConfig()

	opt := WithPublicEndpoints("/healthz", "/readyz", "/metrics")
	opt(cfg)

	if len(cfg.publicEndpoints) != 3 {
		t.Errorf("expected 3 public endpoints, got %d", len(cfg.publicEndpoints))
	}
}

func TestWithHealthCheck(t *testing.T) {
	cfg := newServerConfig()

	if cfg.healthEnabled {
		t.Error("expected health to be disabled by default")
	}

	opt := WithHealthCheck()
	opt(cfg)

	if !cfg.healthEnabled {
		t.Error("expected health to be enabled")
	}
}

func TestWithMetrics(t *testing.T) {
	cfg := newServerConfig()

	if cfg.metricsEnabled {
		t.Error("expected metrics to be disabled by default")
	}

	opt := WithMetrics()
	opt(cfg)

	if !cfg.metricsEnabled {
		t.Error("expected metrics to be enabled")
	}
}

func TestWithCORS(t *testing.T) {
	cfg := newServerConfig()

	if cfg.corsEnabled {
		t.Error("expected CORS to be disabled by default")
	}

	opt := WithCORS()
	opt(cfg)

	if !cfg.corsEnabled {
		t.Error("expected CORS to be enabled")
	}

	if cfg.corsConfig == nil {
		t.Error("expected CORS config to be set")
	}

	// Should have default config
	if len(cfg.corsConfig.AllowedOrigins) == 0 || cfg.corsConfig.AllowedOrigins[0] != "*" {
		t.Error("expected default CORS config with wildcard origin")
	}
}

func TestWithCORSConfig(t *testing.T) {
	cfg := newServerConfig()

	customConfig := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	opt := WithCORSConfig(customConfig)
	opt(cfg)

	if !cfg.corsEnabled {
		t.Error("expected CORS to be enabled")
	}

	if cfg.corsConfig.AllowedOrigins[0] != "https://example.com" {
		t.Errorf("expected custom origin, got %v", cfg.corsConfig.AllowedOrigins)
	}
}

func TestWithSwagger(t *testing.T) {
	cfg := newServerConfig()

	opt := WithSwagger("https://example.com/swagger.json")
	opt(cfg)

	if !cfg.swaggerEnabled {
		t.Error("expected swagger to be enabled")
	}

	if cfg.swaggerURL != "https://example.com/swagger.json" {
		t.Errorf("expected swagger URL, got %s", cfg.swaggerURL)
	}
}

func TestWithSwaggerFile(t *testing.T) {
	cfg := newServerConfig()

	opt := WithSwaggerFile("./api/swagger.json")
	opt(cfg)

	if !cfg.swaggerEnabled {
		t.Error("expected swagger to be enabled")
	}

	if cfg.swaggerPath != "./api/swagger.json" {
		t.Errorf("expected swagger path, got %s", cfg.swaggerPath)
	}
}

func TestWithMarshaler(t *testing.T) {
	cfg := newServerConfig()

	marshaler := &FormMarshaler{}
	opt := WithMarshaler("application/x-www-form-urlencoded", marshaler)
	opt(cfg)

	if cfg.marshalers["application/x-www-form-urlencoded"] != marshaler {
		t.Error("expected marshaler to be registered")
	}
}

func TestWithMarshalers(t *testing.T) {
	cfg := newServerConfig()

	marshalers := map[string]runtime.Marshaler{
		"application/xml":  &XMLMarshaler{},
		"text/plain":       &TextMarshaler{},
	}

	opt := WithMarshalers(marshalers)
	opt(cfg)

	if len(cfg.marshalers) != 2 {
		t.Errorf("expected 2 marshalers, got %d", len(cfg.marshalers))
	}
}

func TestWithJSONOptions(t *testing.T) {
	cfg := newServerConfig()

	opts := JSONOptions{
		UseProtoNames:   true,
		EmitUnpopulated: true,
		Indent:          "  ",
		DiscardUnknown:  true,
	}

	opt := WithJSONOptions(opts)
	opt(cfg)

	if cfg.jsonOptions == nil {
		t.Error("expected JSON options to be set")
	}

	if !cfg.jsonOptions.UseProtoNames {
		t.Error("expected UseProtoNames to be true")
	}
}

func TestWithGracefulShutdown(t *testing.T) {
	cfg := newServerConfig()

	opt := WithGracefulShutdown(60 * time.Second)
	opt(cfg)

	if cfg.gracefulTimeout != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", cfg.gracefulTimeout)
	}
}

func TestWithLogLevel(t *testing.T) {
	cfg := newServerConfig()

	opt := WithLogLevel("debug")
	opt(cfg)

	if cfg.logLevel != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.logLevel)
	}
}

func TestWithHTTPHandler(t *testing.T) {
	cfg := newServerConfig()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	opt := WithHTTPHandler("/webhook", handler)
	opt(cfg)

	if len(cfg.httpHandlers) != 1 {
		t.Errorf("expected 1 HTTP handler, got %d", len(cfg.httpHandlers))
	}

	if cfg.httpHandlers[0].pattern != "/webhook" {
		t.Errorf("expected pattern /webhook, got %s", cfg.httpHandlers[0].pattern)
	}
}

func TestWithHTTPHandlerFunc(t *testing.T) {
	cfg := newServerConfig()

	opt := WithHTTPHandlerFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	opt(cfg)

	if len(cfg.httpHandlers) != 1 {
		t.Errorf("expected 1 HTTP handler, got %d", len(cfg.httpHandlers))
	}
}

func TestWithHTTPMiddleware(t *testing.T) {
	cfg := newServerConfig()

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	opt := WithHTTPMiddleware(middleware)
	opt(cfg)

	if len(cfg.httpMiddlewares) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(cfg.httpMiddlewares))
	}
}

func TestWithUnaryInterceptor(t *testing.T) {
	cfg := newServerConfig()

	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	}

	opt := WithUnaryInterceptor(interceptor)
	opt(cfg)

	if len(cfg.unaryInterceptors) != 1 {
		t.Errorf("expected 1 unary interceptor, got %d", len(cfg.unaryInterceptors))
	}
}

func TestWithUnaryInterceptor_WithExceptions(t *testing.T) {
	cfg := newServerConfig()

	interceptor := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		return handler(ctx, req)
	}

	opt := WithUnaryInterceptor(interceptor, ExceptEndpoints("/test.Service/SkipThis", "/test.Service/SkipThat"))
	opt(cfg)

	if len(cfg.unaryInterceptors) != 1 {
		t.Errorf("expected 1 unary interceptor, got %d", len(cfg.unaryInterceptors))
	}

	if len(cfg.unaryInterceptors[0].exceptEndpoints) != 2 {
		t.Errorf("expected 2 except endpoints, got %d", len(cfg.unaryInterceptors[0].exceptEndpoints))
	}
}

func TestWithStreamInterceptor(t *testing.T) {
	cfg := newServerConfig()

	interceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	opt := WithStreamInterceptor(interceptor)
	opt(cfg)

	if len(cfg.streamInterceptors) != 1 {
		t.Errorf("expected 1 stream interceptor, got %d", len(cfg.streamInterceptors))
	}
}

func TestWithStreamInterceptor_WithExceptions(t *testing.T) {
	cfg := newServerConfig()

	interceptor := func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	opt := WithStreamInterceptor(interceptor, ExceptEndpoints("/test.Service/StreamMethod"))
	opt(cfg)

	if len(cfg.streamInterceptors[0].exceptEndpoints) != 1 {
		t.Errorf("expected 1 except endpoint, got %d", len(cfg.streamInterceptors[0].exceptEndpoints))
	}
}

func TestExceptEndpoints(t *testing.T) {
	cfg := &interceptorConfig{}

	opt := ExceptEndpoints("/a", "/b", "/c")
	opt(cfg)

	if len(cfg.exceptEndpoints) != 3 {
		t.Errorf("expected 3 except endpoints, got %d", len(cfg.exceptEndpoints))
	}
}

func TestWithGatewayOption(t *testing.T) {
	cfg := newServerConfig()

	// Just test that it appends without error
	opt := WithGatewayOption(runtime.WithMarshalerOption("test", nil))
	opt(cfg)

	if len(cfg.gatewayOptions) != 1 {
		t.Errorf("expected 1 gateway option, got %d", len(cfg.gatewayOptions))
	}
}

func TestMultipleOptionsChaining(t *testing.T) {
	cfg := newServerConfig()

	opts := []Option{
		WithGRPCPort(9091),
		WithHTTPPort(8081),
		WithHealthCheck(),
		WithMetrics(),
		WithCORS(),
		WithGracefulShutdown(45 * time.Second),
		WithLogLevel("debug"),
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.grpcPort != 9091 {
		t.Errorf("expected gRPC port 9091, got %d", cfg.grpcPort)
	}
	if cfg.httpPort != 8081 {
		t.Errorf("expected HTTP port 8081, got %d", cfg.httpPort)
	}
	if !cfg.healthEnabled {
		t.Error("expected health enabled")
	}
	if !cfg.metricsEnabled {
		t.Error("expected metrics enabled")
	}
	if !cfg.corsEnabled {
		t.Error("expected CORS enabled")
	}
	if cfg.gracefulTimeout != 45*time.Second {
		t.Errorf("expected 45s timeout, got %v", cfg.gracefulTimeout)
	}
	if cfg.logLevel != "debug" {
		t.Errorf("expected debug log level, got %s", cfg.logLevel)
	}
}
