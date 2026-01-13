package grpckit

import (
	"context"
	"net/http"
	"path"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// authMiddleware creates HTTP middleware for authentication.
func authMiddleware(cfg *serverConfig, next http.Handler) http.Handler {
	if cfg.authFunc == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this endpoint requires auth
		if !requiresAuth(r.URL.Path, cfg) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		token := extractToken(r.Header.Get("Authorization"))

		// Call auth function
		ctx, err := cfg.authFunc(r.Context(), token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Continue with enriched context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// grpcAuthInterceptor creates a gRPC unary interceptor for authentication.
func grpcAuthInterceptor(cfg *serverConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if cfg.authFunc == nil {
			return handler(ctx, req)
		}

		// Check if this method requires auth
		if !requiresAuth(info.FullMethod, cfg) {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		token := ""
		if len(tokens) > 0 {
			token = extractToken(tokens[0])
		}

		// Call auth function
		newCtx, err := cfg.authFunc(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		return handler(newCtx, req)
	}
}

// grpcStreamAuthInterceptor creates a gRPC stream interceptor for authentication.
func grpcStreamAuthInterceptor(cfg *serverConfig) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if cfg.authFunc == nil {
			return handler(srv, ss)
		}

		// Check if this method requires auth
		if !requiresAuth(info.FullMethod, cfg) {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		token := ""
		if len(tokens) > 0 {
			token = extractToken(tokens[0])
		}

		// Call auth function
		_, err := cfg.authFunc(ctx, token)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		return handler(srv, ss)
	}
}

// requiresAuth checks if a path/method requires authentication.
func requiresAuth(urlPath string, cfg *serverConfig) bool {
	// If protected endpoints are specified, only those require auth
	if len(cfg.protectedEndpoints) > 0 {
		return matchesAnyPattern(urlPath, cfg.protectedEndpoints)
	}

	// If public endpoints are specified, everything except those requires auth
	if len(cfg.publicEndpoints) > 0 {
		return !matchesAnyPattern(urlPath, cfg.publicEndpoints)
	}

	// If auth is set but no patterns, protect everything
	return cfg.authFunc != nil
}

// matchesAnyPattern checks if a path matches any of the glob patterns.
func matchesAnyPattern(urlPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(pattern, urlPath) {
			return true
		}
	}
	return false
}

// matchPattern matches a path against a glob pattern.
// Supports * as wildcard for single path segment and ** for multiple segments.
func matchPattern(pattern, urlPath string) bool {
	// Handle exact match
	if pattern == urlPath {
		return true
	}

	// Handle ** (match any path)
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(urlPath, prefix)
	}

	// Handle * (match single segment)
	if strings.Contains(pattern, "*") {
		matched, _ := path.Match(pattern, urlPath)
		return matched
	}

	return false
}

// extractToken extracts the token from the Authorization header.
// Handles "Bearer <token>" format.
func extractToken(header string) string {
	if header == "" {
		return ""
	}

	// Handle "Bearer <token>"
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}

	// Handle "bearer <token>" (case insensitive)
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return header[7:]
	}

	// Return as-is if no Bearer prefix
	return header
}
