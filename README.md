# gRPCkit

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) 
[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![Open Source Love 
svg1](https://badges.frapsoft.com/os/v1/open-source.svg?v=103)](https://github.com/ellerbrock/open-source-badges/)

![logo](assets/logo.jpg?raw=true)

A dead-simple Go library for building gRPC + REST microservices with [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

## Features

- **gRPC + REST** - Serve both protocols from a single service definition
- **Custom Content Types** - Support for form-urlencoded, XML, binary, multipart, and more
- **Health checks** - Built-in `/healthz` and `/readyz` endpoints
- **Prometheus metrics** - Built-in `/metrics` endpoint
- **Swagger UI** - Serve OpenAPI documentation at `/swagger/`
- **Authentication** - Decorator pattern for protecting endpoints
- **Graceful shutdown** - Clean shutdown with configurable timeout
- **Zero boilerplate** - Focus on your business logic, not infrastructure

## Installation

```bash
go get github.com/gyozatech/grpckit
```

## Quick Start

```go
package main

import (
    "github.com/gyozatech/grpckit"
    pb "your/proto/gen"
    "google.golang.org/grpc"
)

func main() {
    grpckit.Run(
        grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
            pb.RegisterMyServiceServer(s, &MyService{})
        }),
        grpckit.WithRESTService(pb.RegisterMyServiceHandlerFromEndpoint),
    )
}
```

That's it! Your service is now available via:
- **gRPC** on port `9090`
- **REST** on port `8080`

## Multiple Services

Register multiple gRPC services from different proto files.

> **Note**: The function wrapper `func(s grpc.ServiceRegistrar) { ... }` ensures compile-time type checking between your service implementation and the generated interface.

```go
grpckit.Run(
    // Register multiple gRPC services
    grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
        itempb.RegisterItemServiceServer(s, NewItemService())
    }),
    grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
        userpb.RegisterUserServiceServer(s, NewUserService())
    }),
    grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
        orderpb.RegisterOrderServiceServer(s, NewOrderService())
    }),

    // Register their REST handlers
    grpckit.WithRESTService(itempb.RegisterItemServiceHandlerFromEndpoint),
    grpckit.WithRESTService(userpb.RegisterUserServiceHandlerFromEndpoint),
    grpckit.WithRESTService(orderpb.RegisterOrderServiceHandlerFromEndpoint),

    // Configuration applies to all services
    grpckit.WithHealthCheck(),
    grpckit.WithCORS(),
)
```

All services share:
- Same gRPC port (9090) and HTTP port (8080)
- Same interceptors and middleware
- Same authentication configuration
- Same CORS settings

## Configuration

### Functional Options (Recommended)

```go
grpckit.Run(
    // Services
    grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
        pb.RegisterMyServiceServer(s, &MyService{})
    }),
    grpckit.WithRESTService(pb.RegisterMyServiceHandlerFromEndpoint),

    // Ports
    grpckit.WithGRPCPort(9090),
    grpckit.WithHTTPPort(8080),

    // Authentication
    grpckit.WithAuth(myAuthFunc),
    grpckit.WithProtectedEndpoints("/api/v1/admin/*"),
    // OR
    grpckit.WithPublicEndpoints("/healthz", "/readyz", "/metrics"),

    // Features
    grpckit.WithHealthCheck(),
    grpckit.WithMetrics(),
    grpckit.WithSwagger("./api/swagger.json"),

    // Graceful shutdown
    grpckit.WithGracefulShutdown(30 * time.Second),
)
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GRPCKIT_GRPC_PORT` | gRPC server port | `9090` |
| `GRPCKIT_HTTP_PORT` | HTTP/REST server port | `8080` |
| `GRPCKIT_HEALTH_ENABLED` | Enable health endpoints | `false` |
| `GRPCKIT_METRICS_ENABLED` | Enable metrics endpoint | `false` |
| `GRPCKIT_SWAGGER_ENABLED` | Enable Swagger UI | `false` |
| `GRPCKIT_SWAGGER_PATH` | Path to swagger.json | - |
| `GRPCKIT_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `GRPCKIT_GRACEFUL_TIMEOUT` | Shutdown timeout (e.g., "30s") | `30s` |

### YAML Config File

```yaml
# grpckit.yaml
grpc:
  port: 9090
http:
  port: 8080
health:
  enabled: true
metrics:
  enabled: true
swagger:
  enabled: true
  path: "./api/swagger.json"
auth:
  protected_endpoints:
    - "/api/v1/admin/*"
  public_endpoints:
    - "/healthz"
    - "/readyz"
```

Load with:
```go
grpckit.Run(
    grpckit.WithConfigFile("grpckit.yaml"),
    // ... other options override config file
)
```

## Authentication

### Define an Auth Function

```go
authFunc := func(ctx context.Context, token string) (context.Context, error) {
    if token == "" {
        return nil, grpckit.ErrUnauthorized
    }

    // Validate token and extract user info
    userID, err := validateToken(token)
    if err != nil {
        return nil, grpckit.ErrUnauthorized
    }

    // Return enriched context
    return context.WithValue(ctx, "user_id", userID), nil
}
```

### Protect Endpoints

```go
// Option 1: Protect specific endpoints (allowlist)
grpckit.WithAuth(authFunc),
grpckit.WithProtectedEndpoints(
    "/api/v1/users/*",
    "/api/v1/admin/*",
)

// Option 2: Make everything protected except specific endpoints (denylist)
grpckit.WithAuth(authFunc),
grpckit.WithPublicEndpoints(
    "/healthz",
    "/readyz",
    "/metrics",
    "/swagger/*",
)
```

## CORS

Enable Cross-Origin Resource Sharing (CORS) to allow browser requests from different origins.

### Quick Setup

```go
// Enable permissive CORS (allows all origins)
grpckit.WithCORS()
```

### Custom Configuration

```go
grpckit.WithCORSConfig(grpckit.CORSConfig{
    AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           3600, // Cache preflight for 1 hour
})
```

### Default Configuration

When using `WithCORS()`, the default configuration:
- Allows all origins (`*`)
- Allows common HTTP methods (GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD)
- Allows common headers (Authorization, Content-Type, etc.)
- Enables credentials (when not using wildcard origin)
- Caches preflight requests for 24 hours

## Custom Content Types

grpckit supports multiple content types beyond JSON/protobuf. Enable them with simple options.

### Quick Setup

```go
grpckit.Run(
    grpckit.WithGRPCService(...),
    grpckit.WithRESTService(...),

    // Enable form submissions
    grpckit.WithFormURLEncodedSupport(),

    // Enable XML
    grpckit.WithXMLSupport(),

    // Enable binary data
    grpckit.WithBinarySupport(),

    // Enable file uploads
    grpckit.WithMultipartSupport(),

    // Enable plain text
    grpckit.WithTextSupport(),
)
```

### Available Content Types

| Option | Content-Type | Use Case |
|--------|--------------|----------|
| `WithFormURLEncodedSupport()` | `application/x-www-form-urlencoded` | HTML form submissions |
| `WithXMLSupport()` | `application/xml` | Legacy API compatibility |
| `WithBinarySupport()` | `application/octet-stream` | File downloads, raw bytes |
| `WithMultipartSupport()` | `multipart/form-data` | File uploads |
| `WithTextSupport()` | `text/plain` | Plain text endpoints |

### Form URL-Encoded Example

Accept HTML form submissions:

```go
grpckit.WithFormURLEncodedSupport()
```

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=John&email=john@example.com&age=30"
```

Field mapping:
- Uses proto field names (snake_case)
- Nested fields via dot notation: `address.street=123`
- Repeated fields via multiple values: `tags=a&tags=b`

### File Uploads (Multipart)

```go
grpckit.WithMultipartSupport()
```

Define your proto with file fields:
```protobuf
message UploadRequest {
  string description = 1;
  bytes file_data = 2;   // File contents
  string file_name = 3;  // Original filename
  string file_type = 4;  // Content-Type
}
```

Upload files:
```bash
curl -X POST http://localhost:8080/api/v1/upload \
  -F "description=My document" \
  -F "file=@document.pdf"
```

### Custom JSON Options

Configure JSON serialization behavior:

```go
grpckit.WithJSONOptions(grpckit.JSONOptions{
    UseProtoNames:   true,  // Use snake_case instead of camelCase
    EmitUnpopulated: true,  // Include fields with zero values
    Indent:          "  ",  // Pretty print
    DiscardUnknown:  true,  // Ignore unknown fields on input
})
```

### Custom Marshalers

Register your own marshaler for any content type:

```go
grpckit.WithMarshaler("application/msgpack", &MyMsgPackMarshaler{})
```

Or register multiple at once:

```go
grpckit.WithMarshalers(map[string]runtime.Marshaler{
    "application/msgpack": &MyMsgPackMarshaler{},
    "application/yaml":    &MyYAMLMarshaler{},
})
```

### How Content-Type Selection Works

- **Request**: Marshaler selected based on `Content-Type` header
- **Response**: Marshaler selected based on `Accept` header
- **Fallback**: JSON is used when no specific marshaler matches

## Custom HTTP Endpoints

Register HTTP endpoints outside of proto/gRPC. These are pure HTTP handlers that:
- Are NOT exposed via gRPC (HTTP only)
- Can use any input/output format (not constrained by proto)
- Support custom per-handler middleware

### Basic Registration

```go
grpckit.Run(
    grpckit.WithGRPCService(...),
    grpckit.WithRESTService(...),

    // Custom HTTP endpoint (not in proto)
    grpckit.WithHTTPHandler("/webhook", webhookHandler),
    grpckit.WithHTTPHandlerFunc("/upload", uploadFunc),
)
```

### Per-Handler Middleware

Wrap handlers with dedicated middleware:

```go
// Webhook with signature validation middleware
grpckit.WithHTTPHandler("/webhook",
    webhookAuthMiddleware("secret")(
        http.HandlerFunc(webhookHandler),
    ),
),
```

### Global HTTP Middleware

Add middleware that applies to ALL HTTP requests:

```go
grpckit.WithHTTPMiddleware(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("[HTTP] %s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
})
```

### Middleware Execution Order

```
Request
  ↓
metrics middleware (built-in)
  ↓
auth middleware (built-in)
  ↓
custom global middleware(s)
  ↓
per-handler middleware (if wrapped)
  ↓
Handler
```

## gRPC Interceptors

Add custom interceptors for ALL gRPC calls. Interceptors are the gRPC equivalent of HTTP middleware.

### Unary Interceptors

For request-response RPC calls:

```go
grpckit.WithUnaryInterceptor(func(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    log.Printf("[gRPC] %s", info.FullMethod)
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("[gRPC] %s took %v", info.FullMethod, time.Since(start))
    return resp, err
})
```

### Stream Interceptors

For streaming RPC calls:

```go
grpckit.WithStreamInterceptor(func(
    srv interface{},
    ss grpc.ServerStream,
    info *grpc.StreamServerInfo,
    handler grpc.StreamHandler,
) error {
    log.Printf("[gRPC Stream] %s", info.FullMethod)
    return handler(srv, ss)
})
```

### Interceptor Execution Order

```
gRPC Request
  ↓
auth interceptor (built-in, if configured)
  ↓
custom interceptor 1 (first WithUnaryInterceptor call)
  ↓
custom interceptor 2 (second WithUnaryInterceptor call)
  ↓
... more custom interceptors ...
  ↓
Handler
```

### Common Use Cases

- **Logging**: Log method calls, durations, errors
- **Metrics**: Track request counts, latencies
- **Tracing**: Add distributed tracing spans
- **Validation**: Validate requests before handlers
- **Recovery**: Catch panics and convert to errors

### Excluding Endpoints

Skip specific endpoints from an interceptor using `ExceptEndpoints`:

```go
// Timing interceptor that skips high-frequency endpoints
grpckit.WithUnaryInterceptor(timingInterceptor,
    grpckit.ExceptEndpoints(
        "/item.v1.ItemService/HealthCheck",
        "/item.v1.ItemService/ListItems",
    ),
)

// Same for stream interceptors
grpckit.WithStreamInterceptor(streamInterceptor,
    grpckit.ExceptEndpoints("/item.v1.ItemService/StreamItems"),
)
```

Endpoints should be in the format `/package.Service/Method`.

## Endpoints

| Endpoint | Description | Option |
|----------|-------------|--------|
| `/healthz` | Liveness probe (always returns 200 if running) | `WithHealthCheck()` |
| `/readyz` | Readiness probe (returns 503 if not ready) | `WithHealthCheck()` |
| `/metrics` | Prometheus metrics | `WithMetrics()` |
| `/swagger/` | Swagger UI | `WithSwagger(path)` |
| `/swagger/spec.json` | OpenAPI spec | `WithSwagger(path)` |

## Errors

grpckit provides common errors for use in your services:

```go
grpckit.ErrUnauthorized      // 401 - Missing or invalid token
grpckit.ErrForbidden         // 403 - Insufficient permissions
grpckit.ErrNotFound          // 404 - Resource not found
grpckit.ErrInvalidConfig     // Invalid configuration
grpckit.ErrServiceNotRegistered // No services registered
```

## Advanced Usage

### Access Underlying Servers

```go
server, err := grpckit.New(
    grpckit.WithGRPCService(...),
    grpckit.WithRESTService(...),
)
if err != nil {
    log.Fatal(err)
}

// Access gRPC server for advanced configuration
grpcServer := server.GRPCServer()

// Control readiness
server.SetReady(false) // Mark as not ready
server.SetReady(true)  // Mark as ready

// Start the server
server.Start()
```

## Example

See the [example](./example) directory for a complete working example with:
- Proto definitions with REST annotations
- CRUD service implementation
- Authentication
- All features enabled

## License

Apache 2.0
