package main

import (
	"context"
	"log"
	"time"

	"github.com/gyozatech/grpckit"
	"google.golang.org/grpc"
)

// ============================================================
// gRPC Interceptors: Applied to ALL gRPC calls
// ============================================================

// loggingUnaryInterceptor logs all gRPC unary (request-response) calls.
func loggingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.Printf("[gRPC] Start: %s", info.FullMethod)
	resp, err := handler(ctx, req)
	if err != nil {
		log.Printf("[gRPC] Error: %s - %v", info.FullMethod, err)
	} else {
		log.Printf("[gRPC] Done: %s", info.FullMethod)
	}
	return resp, err
}

// timingUnaryInterceptor measures and logs the duration of gRPC calls.
func timingUnaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)
	log.Printf("[gRPC Timing] %s took %v", info.FullMethod, duration)
	return resp, err
}

// loggingStreamInterceptor logs all gRPC streaming calls.
func loggingStreamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Printf("[gRPC Stream] Start: %s", info.FullMethod)
	err := handler(srv, ss)
	if err != nil {
		log.Printf("[gRPC Stream] Error: %s - %v", info.FullMethod, err)
	} else {
		log.Printf("[gRPC Stream] Done: %s", info.FullMethod)
	}
	return err
}

/* not really an interceptor but this function is applied
   by the built-in grpckit.WithAuth() function to the
   grpcAuthInterceptor provided by this library when you natively enable Auth */

// Example authentication function (optional)
func AuthFunc(ctx context.Context, token string) (context.Context, error) {
	if token == "" {
		return nil, grpckit.ErrUnauthorized
	}
	// In a real app, validate the token and extract user info
	return context.WithValue(ctx, "user_id", "user-123"), nil
}
