package grpckit

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// ServiceRegistrar is a function that registers a gRPC service on the server.
// Use this to register your service implementation.
type ServiceRegistrar func(s grpc.ServiceRegistrar)

// RESTRegistrar is a function that registers a REST handler from a gRPC endpoint.
// This matches the signature of generated RegisterXxxHandlerFromEndpoint functions.
type RESTRegistrar func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

// AuthFunc is a function that validates authentication and returns an enriched context.
// The token parameter contains the value from the Authorization header (without "Bearer " prefix).
// Return an error to reject the request, or an enriched context to allow it.
type AuthFunc func(ctx context.Context, token string) (context.Context, error)

// Option is a functional option for configuring the server.
type Option func(*serverConfig)

// serverConfig holds all configuration for the server.
type serverConfig struct {
	// Ports
	grpcPort int
	httpPort int

	// Services
	grpcServices []grpcServiceRegistration
	restServices []RESTRegistrar

	// Authentication
	authFunc           AuthFunc
	protectedEndpoints []string
	publicEndpoints    []string

	// Features
	healthEnabled  bool
	metricsEnabled bool
	swaggerPath    string
	swaggerEnabled bool

	// Content types
	contentTypes map[string]string

	// Shutdown
	gracefulTimeout time.Duration

	// Logging
	logLevel string
}

// grpcServiceRegistration holds a service registrar function.
type grpcServiceRegistration struct {
	registrar ServiceRegistrar
}

// newServerConfig creates a new server config with default values.
func newServerConfig() *serverConfig {
	return &serverConfig{
		grpcPort:        9090,
		httpPort:        8080,
		grpcServices:    make([]grpcServiceRegistration, 0),
		restServices:    make([]RESTRegistrar, 0),
		contentTypes:    make(map[string]string),
		gracefulTimeout: 30 * time.Second,
		logLevel:        "info",
	}
}

// WithGRPCPort sets the gRPC server port.
func WithGRPCPort(port int) Option {
	return func(c *serverConfig) {
		c.grpcPort = port
	}
}

// WithHTTPPort sets the HTTP/REST server port.
func WithHTTPPort(port int) Option {
	return func(c *serverConfig) {
		c.httpPort = port
	}
}

// WithGRPCService registers a gRPC service.
// Pass a function that registers your service on the gRPC server.
//
// Example:
//
//	grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
//	    pb.RegisterMyServiceServer(s, &MyService{})
//	})
func WithGRPCService(registrar ServiceRegistrar) Option {
	return func(c *serverConfig) {
		c.grpcServices = append(c.grpcServices, grpcServiceRegistration{
			registrar: registrar,
		})
	}
}

// WithRESTService registers a REST handler from a gRPC endpoint.
// The registrar should be the generated RegisterXxxHandlerFromEndpoint function from your proto.
//
// Example:
//
//	grpckit.WithRESTService(pb.RegisterMyServiceHandlerFromEndpoint)
func WithRESTService(registrar RESTRegistrar) Option {
	return func(c *serverConfig) {
		c.restServices = append(c.restServices, registrar)
	}
}

// WithAuth sets the authentication function for protected endpoints.
// The function receives the token from the Authorization header and should return
// an enriched context or an error.
//
// Example:
//
//	grpckit.WithAuth(func(ctx context.Context, token string) (context.Context, error) {
//	    if token == "" {
//	        return nil, grpckit.ErrUnauthorized
//	    }
//	    userID := validateToken(token)
//	    return context.WithValue(ctx, "user_id", userID), nil
//	})
func WithAuth(authFunc AuthFunc) Option {
	return func(c *serverConfig) {
		c.authFunc = authFunc
	}
}

// WithProtectedEndpoints sets the endpoints that require authentication.
// Supports glob patterns like "/api/v1/users/*".
// If set, only these endpoints require auth; all others are public.
//
// Example:
//
//	grpckit.WithProtectedEndpoints("/api/v1/users/*", "/api/v1/admin/*")
func WithProtectedEndpoints(patterns ...string) Option {
	return func(c *serverConfig) {
		c.protectedEndpoints = append(c.protectedEndpoints, patterns...)
	}
}

// WithPublicEndpoints sets the endpoints that do NOT require authentication.
// Supports glob patterns like "/healthz".
// If set, all endpoints require auth EXCEPT these.
//
// Example:
//
//	grpckit.WithPublicEndpoints("/healthz", "/readyz", "/metrics")
func WithPublicEndpoints(patterns ...string) Option {
	return func(c *serverConfig) {
		c.publicEndpoints = append(c.publicEndpoints, patterns...)
	}
}

// WithHealthCheck enables health check endpoints (/healthz and /readyz).
func WithHealthCheck() Option {
	return func(c *serverConfig) {
		c.healthEnabled = true
	}
}

// WithMetrics enables the Prometheus metrics endpoint (/metrics).
func WithMetrics() Option {
	return func(c *serverConfig) {
		c.metricsEnabled = true
	}
}

// WithSwagger enables the Swagger UI and serves the OpenAPI spec.
// The path should point to your swagger.json file.
//
// Example:
//
//	grpckit.WithSwagger("./api/swagger.json")
func WithSwagger(swaggerJSONPath string) Option {
	return func(c *serverConfig) {
		c.swaggerEnabled = true
		c.swaggerPath = swaggerJSONPath
	}
}

// WithContentType sets a custom content type for a specific endpoint pattern.
// Use this for endpoints that need something other than application/json.
//
// Example:
//
//	grpckit.WithContentType("/api/v1/upload", "multipart/form-data")
func WithContentType(pattern string, contentType string) Option {
	return func(c *serverConfig) {
		c.contentTypes[pattern] = contentType
	}
}

// WithGracefulShutdown sets the timeout for graceful shutdown.
// Default is 30 seconds.
func WithGracefulShutdown(timeout time.Duration) Option {
	return func(c *serverConfig) {
		c.gracefulTimeout = timeout
	}
}

// WithLogLevel sets the logging level (debug, info, warn, error).
func WithLogLevel(level string) Option {
	return func(c *serverConfig) {
		c.logLevel = level
	}
}
