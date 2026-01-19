package grpckit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Bearer token",
			header:   "Bearer abc123",
			expected: "abc123",
		},
		{
			name:     "bearer lowercase",
			header:   "bearer xyz789",
			expected: "xyz789",
		},
		{
			name:     "BEARER uppercase",
			header:   "BEARER TOKEN123",
			expected: "TOKEN123",
		},
		{
			name:     "no prefix",
			header:   "rawtoken",
			expected: "rawtoken",
		},
		{
			name:     "empty string",
			header:   "",
			expected: "",
		},
		{
			name:     "Bearer with spaces",
			header:   "Bearer   token with spaces",
			expected: "  token with spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToken(tt.header)
			if result != tt.expected {
				t.Errorf("extractToken(%q) = %q, want %q", tt.header, result, tt.expected)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{
			name:     "exact match",
			pattern:  "/api/v1/users",
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name:     "exact no match",
			pattern:  "/api/v1/users",
			path:     "/api/v1/items",
			expected: false,
		},
		{
			name:     "double star suffix",
			pattern:  "/api/v1/**",
			path:     "/api/v1/users/123",
			expected: true,
		},
		{
			name:     "double star no match",
			pattern:  "/api/v1/**",
			path:     "/api/v2/users",
			expected: false,
		},
		{
			name:     "single star",
			pattern:  "/api/v1/*",
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name:     "single star no match nested",
			pattern:  "/api/v1/*",
			path:     "/api/v1/users/123",
			expected: false,
		},
		{
			name:     "grpc method pattern",
			pattern:  "/myservice.v1.MyService/*",
			path:     "/myservice.v1.MyService/GetUser",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPattern(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

func TestMatchesAnyPattern(t *testing.T) {
	patterns := []string{"/healthz", "/readyz", "/api/public/**"}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/healthz", true},
		{"/readyz", true},
		{"/api/public/data", true},
		{"/api/private/data", false},
		{"/metrics", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := matchesAnyPattern(tt.path, patterns)
			if result != tt.expected {
				t.Errorf("matchesAnyPattern(%q, patterns) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestRequiresAuth(t *testing.T) {
	tests := []struct {
		name               string
		path               string
		authFunc           AuthFunc
		protectedEndpoints []string
		publicEndpoints    []string
		expected           bool
	}{
		{
			name:     "no auth func",
			path:     "/api/v1/users",
			authFunc: nil,
			expected: false,
		},
		{
			name:               "protected endpoints - match",
			path:               "/api/v1/users",
			authFunc:           func(ctx context.Context, token string) (context.Context, error) { return ctx, nil },
			protectedEndpoints: []string{"/api/v1/**"},
			expected:           true,
		},
		{
			name:               "protected endpoints - no match",
			path:               "/healthz",
			authFunc:           func(ctx context.Context, token string) (context.Context, error) { return ctx, nil },
			protectedEndpoints: []string{"/api/v1/**"},
			expected:           false,
		},
		{
			name:            "public endpoints - match",
			path:            "/healthz",
			authFunc:        func(ctx context.Context, token string) (context.Context, error) { return ctx, nil },
			publicEndpoints: []string{"/healthz", "/readyz"},
			expected:        false,
		},
		{
			name:            "public endpoints - no match requires auth",
			path:            "/api/v1/users",
			authFunc:        func(ctx context.Context, token string) (context.Context, error) { return ctx, nil },
			publicEndpoints: []string{"/healthz", "/readyz"},
			expected:        true,
		},
		{
			name:     "auth func set, no patterns - protect everything",
			path:     "/anything",
			authFunc: func(ctx context.Context, token string) (context.Context, error) { return ctx, nil },
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &serverConfig{
				authFunc:           tt.authFunc,
				protectedEndpoints: tt.protectedEndpoints,
				publicEndpoints:    tt.publicEndpoints,
			}
			result := requiresAuth(tt.path, cfg)
			if result != tt.expected {
				t.Errorf("requiresAuth(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAuthMiddleware_NoAuthFunc(t *testing.T) {
	cfg := &serverConfig{authFunc: nil}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called when no auth func")
	}
}

func TestAuthMiddleware_PublicEndpoint(t *testing.T) {
	cfg := &serverConfig{
		authFunc:        func(ctx context.Context, token string) (context.Context, error) { return nil, errors.New("auth error") },
		publicEndpoints: []string{"/healthz"},
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called for public endpoint")
	}
}

func TestAuthMiddleware_AuthSuccess(t *testing.T) {
	var capturedCtx context.Context
	cfg := &serverConfig{
		authFunc: func(ctx context.Context, token string) (context.Context, error) {
			if token != "valid-token" {
				return nil, ErrUnauthorized
			}
			return context.WithValue(ctx, "user_id", "user123"), nil
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCtx = r.Context()
		w.WriteHeader(http.StatusOK)
	})

	handler := authMiddleware(cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedCtx.Value("user_id") != "user123" {
		t.Error("expected enriched context with user_id")
	}
}

func TestAuthMiddleware_AuthFailure(t *testing.T) {
	cfg := &serverConfig{
		authFunc: func(ctx context.Context, token string) (context.Context, error) {
			return nil, ErrUnauthorized
		},
	}

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})

	handler := authMiddleware(cfg, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}

	if nextCalled {
		t.Error("next handler should not be called on auth failure")
	}
}

func TestGRPCAuthInterceptor_NoAuthFunc(t *testing.T) {
	cfg := &serverConfig{authFunc: nil}
	interceptor := grpcAuthInterceptor(cfg)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	resp, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should be called when no auth func")
	}
	if resp != "response" {
		t.Errorf("unexpected response: %v", resp)
	}
}

func TestGRPCAuthInterceptor_PublicEndpoint(t *testing.T) {
	cfg := &serverConfig{
		authFunc:        func(ctx context.Context, token string) (context.Context, error) { return nil, errors.New("should not be called") },
		publicEndpoints: []string{"/test.Service/*"},
	}
	interceptor := grpcAuthInterceptor(cfg)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return "response", nil
	}

	_, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler should be called for public endpoint")
	}
}

func TestGRPCAuthInterceptor_MissingMetadata(t *testing.T) {
	cfg := &serverConfig{
		authFunc: func(ctx context.Context, token string) (context.Context, error) {
			return ctx, nil
		},
	}
	interceptor := grpcAuthInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	// Context without metadata
	_, err := interceptor(context.Background(), "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)

	if err == nil {
		t.Error("expected error for missing metadata")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated error, got %v", err)
	}
}

func TestGRPCAuthInterceptor_AuthSuccess(t *testing.T) {
	cfg := &serverConfig{
		authFunc: func(ctx context.Context, token string) (context.Context, error) {
			if token != "valid-token" {
				return nil, ErrUnauthorized
			}
			return context.WithValue(ctx, "user_id", "user123"), nil
		},
	}
	interceptor := grpcAuthInterceptor(cfg)

	var capturedCtx context.Context
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		capturedCtx = ctx
		return "response", nil
	}

	md := metadata.New(map[string]string{"authorization": "Bearer valid-token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if capturedCtx.Value("user_id") != "user123" {
		t.Error("expected enriched context with user_id")
	}
}

func TestGRPCAuthInterceptor_AuthFailure(t *testing.T) {
	cfg := &serverConfig{
		authFunc: func(ctx context.Context, token string) (context.Context, error) {
			return nil, ErrUnauthorized
		},
	}
	interceptor := grpcAuthInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		t.Error("handler should not be called")
		return nil, nil
	}

	md := metadata.New(map[string]string{"authorization": "Bearer invalid-token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, "request", &grpc.UnaryServerInfo{FullMethod: "/test/Method"}, handler)

	if err == nil {
		t.Error("expected authentication error")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("expected Unauthenticated error, got %v", err)
	}
}
