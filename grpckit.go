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
//   - gRPC and REST (via grpc-gateway) on separate ports
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
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
	unaryInterceptors = append(unaryInterceptors, cfg.unaryInterceptors...)
	if len(unaryInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
	}

	// Build stream interceptor chain: auth (if configured) + custom interceptors
	var streamInterceptors []grpc.StreamServerInterceptor
	if cfg.authFunc != nil {
		streamInterceptors = append(streamInterceptors, grpcStreamAuthInterceptor(cfg))
	}
	streamInterceptors = append(streamInterceptors, cfg.streamInterceptors...)
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

	// Start gRPC server
	g.Go(func() error {
		return s.startGRPC()
	})

	// Start HTTP server
	g.Go(func() error {
		return s.startHTTP(ctx)
	})

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
