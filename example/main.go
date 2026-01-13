// Example shows how to use grpckit to build a simple microservice.
//
// This example demonstrates a complete CRUD service for items.
// See item_service.go for the service implementation.
package main

import (
	"context"
	"log"
	"time"

	"github.com/gyozatech/grpckit"
	pb "github.com/gyozatech/grpckit/example/proto/gen"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting grpckit example server...")

	// Create the item service
	//itemService := NewItemService()

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

		// Authentication (optional - comment out to disable)
		grpckit.WithAuth(authFunc),
		grpckit.WithPublicEndpoints(
			"/healthz",
			"/readyz",
			"/metrics",
			"/swagger/*",
			"/api/v1/items",   // Allow public list
			"/api/v1/items/*", // Allow public CRUD for demo
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
