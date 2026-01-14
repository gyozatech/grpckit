// Example shows how to use grpckit to build a simple microservice.
//
// This example demonstrates:
// - A complete CRUD service for items (proto-based, gRPC + REST)
// - Custom content type support (form-urlencoded input, XML output)
// - Custom HTTP endpoints outside proto (webhook with dedicated middleware)
// - Custom gRPC interceptors (logging, timing)
//
// See item_service.go for the service implementation.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gyozatech/grpckit"
	pb "github.com/gyozatech/grpckit/example/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting grpckit example server...")

	// Example authentication function (optional)
	authFunc := func(ctx context.Context, token string) (context.Context, error) {
		if token == "" {
			return nil, grpckit.ErrUnauthorized
		}
		// In a real app, validate the token and extract user info
		return context.WithValue(ctx, "user_id", "user-123"), nil
	}

	// Run the server
	if err := grpckit.Run(
		// Register the gRPC service
		grpckit.WithGRPCService(func(s grpc.ServiceRegistrar) {
			pb.RegisterItemServiceServer(s, NewItemService())
		}),

		// Register the REST service (grpc-gateway)
		grpckit.WithRESTService(pb.RegisterItemServiceHandlerFromEndpoint),

		// Ports
		grpckit.WithGRPCPort(9090),
		grpckit.WithHTTPPort(8080),

		// =====================================================
		// Custom gRPC Interceptors
		// =====================================================
		// These interceptors apply to ALL gRPC calls (unary and streaming).
		// They run AFTER the built-in auth interceptor (if configured).
		// Order: auth → logging → timing (first registered = outermost)
		grpckit.WithUnaryInterceptor(loggingUnaryInterceptor),
		grpckit.WithUnaryInterceptor(timingUnaryInterceptor),
		grpckit.WithStreamInterceptor(loggingStreamInterceptor),

		// =====================================================
		// Custom Content Types
		// =====================================================
		// Enable form-urlencoded input (for PatchItem endpoint)
		grpckit.WithFormURLEncodedSupport(),

		// Enable XML output (for PatchItem endpoint response)
		grpckit.WithXMLSupport(),

		// =====================================================
		// Custom HTTP Endpoint: Webhook (not in proto)
		// =====================================================
		// This endpoint is pure HTTP - not exposed via gRPC.
		// It has its own dedicated middleware (webhookAuthMiddleware)
		// that validates X-Webhook-Signature header.
		grpckit.WithHTTPHandler("/webhook",
			webhookAuthMiddleware("my-webhook-secret")(
				http.HandlerFunc(webhookHandler),
			),
		),

		// Authentication (optional - comment out to disable)
		grpckit.WithAuth(authFunc),
		grpckit.WithPublicEndpoints(
			"/healthz",
			"/readyz",
			"/metrics",
			"/swagger/*",
			"/api/v1/items",   // Allow public list
			"/api/v1/items/*", // Allow public CRUD for demo
			"/webhook",        // Webhook uses its own auth (signature validation)
		),

		// Features
		grpckit.WithHealthCheck(),
		grpckit.WithMetrics(),
		grpckit.WithSwagger("./proto/gen/item.swagger.json"),

		// Graceful shutdown
		grpckit.WithGracefulShutdown(30*time.Second),
	); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
