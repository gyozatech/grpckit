// Package grpckit provides a simple, easy-to-use library for building
// gRPC + REST microservices with grpc-gateway.
//
// # Quick Start
//
// The simplest way to use grpckit:
//
//	func main() {
//	    grpckit.Run(
//	        grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
//	            pb.RegisterMyServiceServer(s, &MyService{})
//	        }),
//	        grpckit.WithRESTService(pb.RegisterMyServiceHandlerFromEndpoint),
//	    )
//	}
//
// # Features
//
// grpckit supports:
//   - gRPC and REST (via grpc-gateway) on separate or same port
//   - Single port mode with automatic h2c multiplexing
//   - Health checks (/healthz, /readyz)
//   - Prometheus metrics (/metrics)
//   - Swagger UI (/swagger/)
//   - Authentication via decorator pattern
//   - Graceful shutdown
//
// # Configuration
//
// Configuration can be provided via:
//   - Functional options (code)
//   - Environment variables (GRPCKIT_*)
//   - YAML config file
package grpckit

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// Server represents the grpckit server instance.
type Server struct {
	cfg           *serverConfig
	grpcServer    *grpc.Server
	httpServer    *http.Server
	healthHandler *healthHandler
	metrics       *Metrics
}

// New creates a new Server with the given options.
func New(opts ...Option) (*Server, error) {
	cfg := newServerConfig()

	// Apply environment variables first (lowest priority)
	applyEnvVars(cfg)

	// Apply options (highest priority)
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate configuration
	if len(cfg.grpcServices) == 0 && len(cfg.restServices) == 0 {
		return nil, ErrServiceNotRegistered
	}

	// Build gRPC server with interceptors
	grpcOpts := []grpc.ServerOption{}

	// Build unary interceptor chain: auth (if configured) + custom interceptors
	var unaryInterceptors []grpc.UnaryServerInterceptor
	if cfg.authFunc != nil {
		unaryInterceptors = append(unaryInterceptors, grpcAuthInterceptor(cfg))
	}
	for _, reg := range cfg.unaryInterceptors {
		unaryInterceptors = append(unaryInterceptors, wrapUnaryInterceptor(reg))
	}
	if len(unaryInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}

	// Build stream interceptor chain: auth (if configured) + custom interceptors
	var streamInterceptors []grpc.StreamServerInterceptor
	if cfg.authFunc != nil {
		streamInterceptors = append(streamInterceptors, grpcStreamAuthInterceptor(cfg))
	}
	for _, reg := range cfg.streamInterceptors {
		streamInterceptors = append(streamInterceptors, wrapStreamInterceptor(reg))
	}
	if len(streamInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(streamInterceptors...))
	}

	grpcServer := grpc.NewServer(grpcOpts...)

	// Register gRPC services
	for _, svc := range cfg.grpcServices {
		svc.registrar(grpcServer)
	}

	// Enable reflection for grpcurl/grpcui
	reflection.Register(grpcServer)

	// Create health handler
	healthHandler := newHealthHandler()

	// Create metrics if enabled
	var metrics *Metrics
	if cfg.metricsEnabled {
		metrics = newMetrics("grpckit")
	}

	return &Server{
		cfg:           cfg,
		grpcServer:    grpcServer,
		healthHandler: healthHandler,
		metrics:       metrics,
	}, nil
}

// Run creates and starts a server with the given options.
// This is the simplest way to start a grpckit server.
// It blocks until the server is stopped (via signal or error).
func Run(opts ...Option) error {
	server, err := New(opts...)
	if err != nil {
		return err
	}
	return server.Start()
}

// Start starts the gRPC and HTTP servers.
// It blocks until the server is stopped.
func (s *Server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	g, ctx := errgroup.WithContext(ctx)

	// Check if same-port mode (gRPC and HTTP on same port)
	if s.cfg.grpcPort == s.cfg.httpPort {
		// Same-port mode: use h2c multiplexing
		g.Go(func() error {
			return s.startCombined(ctx)
		})
	} else {
		// Separate ports mode: start each server independently
		g.Go(func() error {
			return s.startGRPC()
		})

		g.Go(func() error {
			return s.startHTTP(ctx)
		})
	}

	// Wait for shutdown signal
	g.Go(func() error {
		select {
		case sig := <-sigCh:
			log.Printf("Received signal %v, shutting down...", sig)
			s.Shutdown()
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	return g.Wait()
}

// startGRPC starts the gRPC server.
func (s *Server) startGRPC() error {
	addr := fmt.Sprintf(":%d", s.cfg.grpcPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	log.Printf("gRPC server listening on %s", addr)
	return s.grpcServer.Serve(lis)
}

// startHTTP starts the HTTP/REST server with grpc-gateway.
func (s *Server) startHTTP(ctx context.Context) error {
	// Create grpc-gateway mux with marshaler options
	gwMux := runtime.NewServeMux(buildMarshalerOptions(s.cfg)...)

	// Register REST services via grpc-gateway
	grpcEndpoint := fmt.Sprintf("localhost:%d", s.cfg.grpcPort)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	for _, registrar := range s.cfg.restServices {
		if err := registrar(ctx, gwMux, grpcEndpoint, opts); err != nil {
			return fmt.Errorf("failed to register REST service: %w", err)
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

	// Register swagger endpoints
	if s.cfg.swaggerEnabled && s.cfg.swaggerPath != "" {
		if err := registerSwaggerEndpoints(mux, s.cfg.swaggerPath); err != nil {
			log.Printf("Warning: failed to register Swagger endpoints: %v", err)
		}
	}

	// Register custom HTTP handlers (before grpc-gateway catch-all)
	for _, h := range s.cfg.httpHandlers {
		mux.Handle(h.pattern, h.handler)
	}

	// Mount grpc-gateway mux for all other paths (catch-all)
	mux.Handle("/", gwMux)

	// Build middleware chain (applied to ALL HTTP requests)
	var handler http.Handler = mux

	// Apply custom HTTP middlewares (in reverse order so first registered = outermost)
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

	// Apply built-in CORS middleware (outermost, handles preflight OPTIONS)
	if s.cfg.corsEnabled && s.cfg.corsConfig != nil {
		handler = corsMiddleware(*s.cfg.corsConfig)(handler)
	}

	// Create HTTP server
	addr := fmt.Sprintf(":%d", s.cfg.httpPort)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("HTTP server listening on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// startCombined starts a combined gRPC + HTTP server on a single port using h2c.
// This allows both gRPC and REST to be served on the same port.
func (s *Server) startCombined(ctx context.Context) error {
	// Build the HTTP handler (same as startHTTP)
	gwMux := runtime.NewServeMux(buildMarshalerOptions(s.cfg)...)

	// Register REST services via grpc-gateway
	// In combined mode, we connect to ourselves via the same port
	grpcEndpoint := fmt.Sprintf("localhost:%d", s.cfg.grpcPort)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	for _, registrar := range s.cfg.restServices {
		if err := registrar(ctx, gwMux, grpcEndpoint, opts); err != nil {
			return fmt.Errorf("failed to register REST service: %w", err)
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

	// Register swagger endpoints
	if s.cfg.swaggerEnabled && s.cfg.swaggerPath != "" {
		if err := registerSwaggerEndpoints(mux, s.cfg.swaggerPath); err != nil {
			log.Printf("Warning: failed to register Swagger endpoints: %v", err)
		}
	}

	// Register custom HTTP handlers (before grpc-gateway catch-all)
	for _, h := range s.cfg.httpHandlers {
		mux.Handle(h.pattern, h.handler)
	}

	// Mount grpc-gateway mux for all other paths (catch-all)
	mux.Handle("/", gwMux)

	// Build middleware chain (applied to ALL HTTP requests)
	var httpHandler http.Handler = mux

	// Apply custom HTTP middlewares (in reverse order so first registered = outermost)
	for i := len(s.cfg.httpMiddlewares) - 1; i >= 0; i-- {
		httpHandler = s.cfg.httpMiddlewares[i](httpHandler)
	}

	// Apply built-in auth middleware
	if s.cfg.authFunc != nil {
		httpHandler = authMiddleware(s.cfg, httpHandler)
	}

	// Apply built-in metrics middleware
	if s.cfg.metricsEnabled && s.metrics != nil {
		httpHandler = metricsMiddleware(s.metrics, httpHandler)
	}

	// Apply built-in CORS middleware (outermost, handles preflight OPTIONS)
	if s.cfg.corsEnabled && s.cfg.corsConfig != nil {
		httpHandler = corsMiddleware(*s.cfg.corsConfig)(httpHandler)
	}

	// Create a combined handler that routes gRPC and HTTP requests
	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a gRPC request
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			s.grpcServer.ServeHTTP(w, r)
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})

	// Wrap with h2c handler for HTTP/2 cleartext support
	h2cHandler := h2c.NewHandler(combinedHandler, &http2.Server{})

	// Create HTTP server
	addr := fmt.Sprintf(":%d", s.cfg.grpcPort)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: h2cHandler,
	}

	log.Printf("gRPC + HTTP server listening on %s (combined mode)", addr)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	// Mark as not ready
	s.healthHandler.SetReady(false)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.gracefulTimeout)
	defer cancel()

	// Shutdown HTTP server
	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
	}

	// Gracefully stop gRPC server
	s.grpcServer.GracefulStop()

	log.Println("Server stopped")
}

// SetReady sets the readiness state of the server.
// Use this to temporarily mark the server as not ready during maintenance.
func (s *Server) SetReady(ready bool) {
	s.healthHandler.SetReady(ready)
}

// GRPCServer returns the underlying gRPC server.
// Use this for advanced configuration or testing.
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// HTTPServer returns the underlying HTTP server.
// Use this for advanced configuration or testing.
// Note: This is only available after Start() is called.
func (s *Server) HTTPServer() *http.Server {
	return s.httpServer
}

// wrapUnaryInterceptor wraps an interceptor with endpoint exclusion logic.
func wrapUnaryInterceptor(reg unaryInterceptorRegistration) grpc.UnaryServerInterceptor {
	if len(reg.exceptEndpoints) == 0 {
		return reg.interceptor
	}
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		for _, endpoint := range reg.exceptEndpoints {
			if info.FullMethod == endpoint {
				return handler(ctx, req) // Skip interceptor
			}
		}
		return reg.interceptor(ctx, req, info, handler)
	}
}

// wrapStreamInterceptor wraps a stream interceptor with endpoint exclusion logic.
func wrapStreamInterceptor(reg streamInterceptorRegistration) grpc.StreamServerInterceptor {
	if len(reg.exceptEndpoints) == 0 {
		return reg.interceptor
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		for _, endpoint := range reg.exceptEndpoints {
			if info.FullMethod == endpoint {
				return handler(srv, ss) // Skip interceptor
			}
		}
		return reg.interceptor(srv, ss, info, handler)
	}
}
