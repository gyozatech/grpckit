package grpckit

import (
	"context"
	"net/http"
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

// HTTPMiddleware is a function that wraps an http.Handler.
// Use this to create middleware for HTTP endpoints.
type HTTPMiddleware func(http.Handler) http.Handler

// InterceptorOption configures an interceptor's behavior.
type InterceptorOption func(*interceptorConfig)

// interceptorConfig holds configuration for an interceptor.
type interceptorConfig struct {
	exceptEndpoints []string
}

// ExceptEndpoints excludes the specified gRPC methods from this interceptor.
// Methods should be in the format "/package.Service/Method".
//
// Example:
//
//	grpckit.WithUnaryInterceptor(timingInterceptor,
//	    grpckit.ExceptEndpoints("/item.v1.ItemService/CreateItem"),
//	)
func ExceptEndpoints(endpoints ...string) InterceptorOption {
	return func(c *interceptorConfig) {
		c.exceptEndpoints = append(c.exceptEndpoints, endpoints...)
	}
}

// httpHandlerRegistration holds a custom HTTP handler registration.
type httpHandlerRegistration struct {
	pattern string
	handler http.Handler
}

// unaryInterceptorRegistration holds a unary interceptor with its config.
type unaryInterceptorRegistration struct {
	interceptor     grpc.UnaryServerInterceptor
	exceptEndpoints []string
}

// streamInterceptorRegistration holds a stream interceptor with its config.
type streamInterceptorRegistration struct {
	interceptor     grpc.StreamServerInterceptor
	exceptEndpoints []string
}

// JSONOptions configures JSON marshaling behavior.
type JSONOptions struct {
	// UseProtoNames uses proto field names (snake_case) instead of camelCase
	UseProtoNames bool

	// EmitUnpopulated emits fields with zero values
	EmitUnpopulated bool

	// Indent sets indentation for pretty printing (empty = compact)
	Indent string

	// DiscardUnknown ignores unknown fields during unmarshaling
	DiscardUnknown bool
}

// globalSwaggerData is set by generated init() code from swagger_gen.go.
// This allows the swagger data to be embedded without user-visible //go:embed.
var globalSwaggerData []byte

// SetSwaggerData is called by the generated swagger_gen.go init() function.
// Users should not call this directly.
func SetSwaggerData(data []byte) {
	globalSwaggerData = data
}

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
	swaggerURL     string // URL for documentation (fetched at build time)
	swaggerPath    string // Local file path (read at runtime)
	swaggerEnabled bool
	corsEnabled    bool
	corsConfig     *CORSConfig

	// Marshalers for custom content types
	marshalers     map[string]runtime.Marshaler
	jsonOptions    *JSONOptions
	gatewayOptions []runtime.ServeMuxOption

	// Custom HTTP handlers (not in proto)
	httpHandlers []httpHandlerRegistration

	// Custom HTTP middleware (applied to ALL HTTP requests)
	httpMiddlewares []HTTPMiddleware

	// Custom gRPC interceptors (applied to ALL gRPC calls)
	unaryInterceptors  []unaryInterceptorRegistration
	streamInterceptors []streamInterceptorRegistration

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
		grpcPort:           9090,
		httpPort:           8080,
		grpcServices:       make([]grpcServiceRegistration, 0),
		restServices:       make([]RESTRegistrar, 0),
		marshalers:         make(map[string]runtime.Marshaler),
		gatewayOptions:     make([]runtime.ServeMuxOption, 0),
		httpHandlers:       make([]httpHandlerRegistration, 0),
		httpMiddlewares:    make([]HTTPMiddleware, 0),
		unaryInterceptors:  make([]unaryInterceptorRegistration, 0),
		streamInterceptors: make([]streamInterceptorRegistration, 0),
		gracefulTimeout:    30 * time.Second,
		logLevel:           "info",
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

// WithCORS enables CORS (Cross-Origin Resource Sharing) with a permissive
// default configuration that allows requests from any origin.
// This is suitable for development and public APIs.
//
// For custom CORS configuration, use WithCORSConfig instead.
//
// Example:
//
//	grpckit.WithCORS()
func WithCORS() Option {
	return func(c *serverConfig) {
		c.corsEnabled = true
		cfg := DefaultCORSConfig()
		c.corsConfig = &cfg
	}
}

// WithCORSConfig enables CORS with a custom configuration.
// Use this for fine-grained control over allowed origins, methods, and headers.
//
// Example:
//
//	grpckit.WithCORSConfig(grpckit.CORSConfig{
//	    AllowedOrigins: []string{"https://example.com", "https://app.example.com"},
//	    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowedHeaders: []string{"Authorization", "Content-Type"},
//	    AllowCredentials: true,
//	    MaxAge: 3600,
//	})
func WithCORSConfig(cfg CORSConfig) Option {
	return func(c *serverConfig) {
		c.corsEnabled = true
		c.corsConfig = &cfg
	}
}

// WithSwagger enables Swagger UI with a URL-based swagger spec.
// The URL is fetched at build time via 'make swagger' and embedded into the binary.
// At runtime, the swagger is served from memory.
//
// To use this:
//  1. Pass the URL to your swagger.json file
//  2. Run 'make swagger' before 'go build' (or just 'make build')
//  3. The Makefile fetches the URL and generates swagger_gen.go
//
// If 'make swagger' wasn't run, /swagger/ returns 404 with a helpful message.
//
// Example:
//
//	grpckit.WithSwagger("https://git.example.com/org/api/-/raw/v1.0.0/swagger.json")
func WithSwagger(url string) Option {
	return func(c *serverConfig) {
		c.swaggerEnabled = true
		c.swaggerURL = url
	}
}

// WithSwaggerFile enables Swagger UI from a local file (read at runtime).
// Use this if you don't need build-time embedding and the file is available at runtime.
//
// Example:
//
//	grpckit.WithSwaggerFile("./api/swagger.json")
func WithSwaggerFile(path string) Option {
	return func(c *serverConfig) {
		c.swaggerEnabled = true
		c.swaggerPath = path
	}
}

// WithMarshaler registers a custom marshaler for a specific MIME type.
// The marshaler handles both request parsing and response formatting.
// Content-Type header determines which marshaler is used for requests,
// and Accept header determines which is used for responses.
//
// Example:
//
//	grpckit.WithMarshaler("application/msgpack", &MyMsgPackMarshaler{})
func WithMarshaler(mimeType string, marshaler runtime.Marshaler) Option {
	return func(c *serverConfig) {
		if c.marshalers == nil {
			c.marshalers = make(map[string]runtime.Marshaler)
		}
		c.marshalers[mimeType] = marshaler
	}
}

// WithMarshalers registers multiple marshalers at once.
// Convenience method for registering multiple content types.
//
// Example:
//
//	grpckit.WithMarshalers(map[string]runtime.Marshaler{
//	    "application/xml": &grpckit.XMLMarshaler{},
//	    "text/plain":      &grpckit.TextMarshaler{},
//	})
func WithMarshalers(marshalers map[string]runtime.Marshaler) Option {
	return func(c *serverConfig) {
		if c.marshalers == nil {
			c.marshalers = make(map[string]runtime.Marshaler)
		}
		for mimeType, marshaler := range marshalers {
			c.marshalers[mimeType] = marshaler
		}
	}
}

// WithJSONOptions configures the default JSON marshaler with custom options.
// Use this to customize proto-to-JSON conversion behavior.
//
// Example:
//
//	grpckit.WithJSONOptions(grpckit.JSONOptions{
//	    UseProtoNames:   true,  // Use snake_case instead of camelCase
//	    EmitUnpopulated: true,  // Include fields with zero values
//	    Indent:          "  ",  // Pretty print with 2-space indent
//	})
func WithJSONOptions(opts JSONOptions) Option {
	return func(c *serverConfig) {
		c.jsonOptions = &opts
	}
}

// WithGatewayOption allows passing raw grpc-gateway ServeMuxOptions.
// Use this for advanced customization not covered by other options.
//
// Example:
//
//	grpckit.WithGatewayOption(runtime.WithMarshalerOption(
//	    "application/json+pretty",
//	    &runtime.JSONPb{...},
//	))
func WithGatewayOption(opt runtime.ServeMuxOption) Option {
	return func(c *serverConfig) {
		c.gatewayOptions = append(c.gatewayOptions, opt)
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

// WithHTTPHandler registers a custom HTTP handler for a URL pattern.
// These handlers bypass grpc-gateway and are NOT exposed via gRPC.
// Use for webhooks, file uploads, or any endpoint that doesn't fit the proto model.
//
// Handlers go through the global middleware chain (auth, metrics, custom middleware).
// For handler-specific middleware, wrap the handler before registering:
//
//	grpckit.WithHTTPHandler("/webhook",
//	    myWebhookMiddleware(http.HandlerFunc(webhookHandler)),
//	)
//
// Example:
//
//	grpckit.WithHTTPHandler("/webhook", webhookHandler)
func WithHTTPHandler(pattern string, handler http.Handler) Option {
	return func(c *serverConfig) {
		c.httpHandlers = append(c.httpHandlers, httpHandlerRegistration{
			pattern: pattern,
			handler: handler,
		})
	}
}

// WithHTTPHandlerFunc registers a custom HTTP handler function for a URL pattern.
// This is a convenience wrapper around WithHTTPHandler for http.HandlerFunc.
//
// Example:
//
//	grpckit.WithHTTPHandlerFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
//	    // Handle webhook - any input/output format
//	    body, _ := io.ReadAll(r.Body)
//	    log.Printf("Webhook: %s", body)
//	    w.Write([]byte("OK"))
//	})
func WithHTTPHandlerFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) Option {
	return WithHTTPHandler(pattern, http.HandlerFunc(handler))
}

// WithHTTPMiddleware adds a middleware to the HTTP middleware chain.
// Middleware is applied to ALL HTTP requests (grpc-gateway, custom handlers, health, etc.)
// Middleware is applied in the order registered (first registered = outermost).
//
// For handler-specific middleware, wrap the handler before registering with WithHTTPHandler.
//
// Example:
//
//	// Global logging middleware
//	grpckit.WithHTTPMiddleware(func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)
//	        next.ServeHTTP(w, r)
//	    })
//	})
func WithHTTPMiddleware(middleware HTTPMiddleware) Option {
	return func(c *serverConfig) {
		c.httpMiddlewares = append(c.httpMiddlewares, middleware)
	}
}

// WithUnaryInterceptor adds a unary interceptor to the gRPC server.
// Interceptors are applied to ALL gRPC unary (request-response) calls.
// Interceptors are applied in the order registered (first registered = outermost).
//
// The built-in auth interceptor (if configured) runs before custom interceptors.
//
// Optional InterceptorOption parameters can configure the interceptor's behavior,
// such as excluding specific endpoints.
//
// Example:
//
//	// Logging interceptor for all endpoints
//	grpckit.WithUnaryInterceptor(func(
//	    ctx context.Context,
//	    req interface{},
//	    info *grpc.UnaryServerInfo,
//	    handler grpc.UnaryHandler,
//	) (interface{}, error) {
//	    log.Printf("[gRPC] %s", info.FullMethod)
//	    return handler(ctx, req)
//	})
//
//	// Timing interceptor that skips specific endpoints
//	grpckit.WithUnaryInterceptor(timingInterceptor,
//	    grpckit.ExceptEndpoints("/item.v1.ItemService/ListItems"),
//	)
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor, opts ...InterceptorOption) Option {
	return func(c *serverConfig) {
		cfg := &interceptorConfig{}
		for _, opt := range opts {
			opt(cfg)
		}
		c.unaryInterceptors = append(c.unaryInterceptors, unaryInterceptorRegistration{
			interceptor:     interceptor,
			exceptEndpoints: cfg.exceptEndpoints,
		})
	}
}

// WithStreamInterceptor adds a stream interceptor to the gRPC server.
// Interceptors are applied to ALL gRPC streaming calls.
// Interceptors are applied in the order registered (first registered = outermost).
//
// The built-in auth interceptor (if configured) runs before custom interceptors.
//
// Optional InterceptorOption parameters can configure the interceptor's behavior,
// such as excluding specific endpoints.
//
// Example:
//
//	// Logging interceptor for all streams
//	grpckit.WithStreamInterceptor(func(
//	    srv interface{},
//	    ss grpc.ServerStream,
//	    info *grpc.StreamServerInfo,
//	    handler grpc.StreamHandler,
//	) error {
//	    log.Printf("[gRPC Stream] %s", info.FullMethod)
//	    return handler(srv, ss)
//	})
//
//	// Stream interceptor that skips specific endpoints
//	grpckit.WithStreamInterceptor(streamInterceptor,
//	    grpckit.ExceptEndpoints("/item.v1.ItemService/StreamItems"),
//	)
func WithStreamInterceptor(interceptor grpc.StreamServerInterceptor, opts ...InterceptorOption) Option {
	return func(c *serverConfig) {
		cfg := &interceptorConfig{}
		for _, opt := range opts {
			opt(cfg)
		}
		c.streamInterceptors = append(c.streamInterceptors, streamInterceptorRegistration{
			interceptor:     interceptor,
			exceptEndpoints: cfg.exceptEndpoints,
		})
	}
}
