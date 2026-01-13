# gRPCkit

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) 
[![made-with-Go](https://img.shields.io/badge/Made%20with-Go-1f425f.svg)](http://golang.org)
[![Open Source Love 
svg1](https://badges.frapsoft.com/os/v1/open-source.svg?v=103)](https://github.com/ellerbrock/open-source-badges/)

![logo](assets/logo.jpg?raw=true)

A dead-simple Go library for building gRPC + REST microservices with [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

## Features

- **gRPC + REST** - Serve both protocols from a single service definition
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
