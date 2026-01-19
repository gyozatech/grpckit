package grpckit

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/grpc"
)

// mockService is a simple mock gRPC service for testing.
type mockService struct{}

func (m *mockService) registerGRPC(s grpc.ServiceRegistrar) {
	// In real tests, you would register your actual service here:
	// pb.RegisterMyServiceServer(s, m)
}

func TestNewTestServer(t *testing.T) {
	// Create a test server with health checks enabled
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {
			// Register a mock service (in real tests, use your actual service)
		}),
		WithHealthCheck(),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Verify the test server was created
	if ts.httpServer == nil {
		t.Error("httpServer should not be nil")
	}
	if ts.grpcListener == nil {
		t.Error("grpcListener should not be nil")
	}
}

func TestTestServer_HTTPClient(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Test health endpoint via HTTP
	resp, err := ts.HTTPClient().Get(ts.URL("/healthz"))
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("GET /healthz body = %s, want to contain 'ok'", body)
	}
}

func TestTestServer_Readiness(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Test readiness endpoint
	resp, err := ts.HTTPClient().Get(ts.URL("/readyz"))
	if err != nil {
		t.Fatalf("GET /readyz error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /readyz status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Set not ready
	ts.SetReady(false)

	resp2, err := ts.HTTPClient().Get(ts.URL("/readyz"))
	if err != nil {
		t.Fatalf("GET /readyz (not ready) error = %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz (not ready) status = %d, want %d", resp2.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestTestServer_URL(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Test URL helper
	url := ts.URL("/api/v1/items")
	if !strings.HasPrefix(url, "http://") {
		t.Errorf("URL should start with http://, got %s", url)
	}
	if !strings.HasSuffix(url, "/api/v1/items") {
		t.Errorf("URL should end with /api/v1/items, got %s", url)
	}

	// Test URL without leading slash
	url2 := ts.URL("api/v1/items")
	if !strings.HasSuffix(url2, "/api/v1/items") {
		t.Errorf("URL should handle missing leading slash, got %s", url2)
	}
}

func TestTestServer_GRPCClientConn(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Get gRPC connection
	conn := ts.GRPCClientConn(context.Background())
	if conn == nil {
		t.Error("GRPCClientConn should return a connection")
	}

	// Verify connection is reused
	conn2 := ts.GRPCClientConn(context.Background())
	if conn != conn2 {
		t.Error("GRPCClientConn should return the same connection")
	}
}

func TestTestServer_WithAuth(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
		WithAuth(MockAuthFunc("secret-token", "user-123")),
		WithPublicEndpoints("/healthz", "/readyz"),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Public endpoint should work without auth
	resp, err := ts.HTTPClient().Get(ts.URL("/healthz"))
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /healthz (public) status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestMockAuthFunc(t *testing.T) {
	authFunc := MockAuthFunc("valid-token", "user-123")

	// Test valid token
	ctx, err := authFunc(context.Background(), "valid-token")
	if err != nil {
		t.Errorf("MockAuthFunc with valid token error = %v", err)
	}
	if ctx.Value("user_id") != "user-123" {
		t.Errorf("MockAuthFunc user_id = %v, want user-123", ctx.Value("user_id"))
	}

	// Test invalid token
	_, err = authFunc(context.Background(), "invalid-token")
	if err != ErrUnauthorized {
		t.Errorf("MockAuthFunc with invalid token error = %v, want ErrUnauthorized", err)
	}
}

func TestMockAuthFuncMultiple(t *testing.T) {
	authFunc := MockAuthFuncMultiple(map[string]string{
		"admin-token": "admin-user",
		"user-token":  "regular-user",
	})

	// Test admin token
	ctx, err := authFunc(context.Background(), "admin-token")
	if err != nil {
		t.Errorf("MockAuthFuncMultiple with admin token error = %v", err)
	}
	if ctx.Value("user_id") != "admin-user" {
		t.Errorf("MockAuthFuncMultiple user_id = %v, want admin-user", ctx.Value("user_id"))
	}

	// Test user token
	ctx, err = authFunc(context.Background(), "user-token")
	if err != nil {
		t.Errorf("MockAuthFuncMultiple with user token error = %v", err)
	}
	if ctx.Value("user_id") != "regular-user" {
		t.Errorf("MockAuthFuncMultiple user_id = %v, want regular-user", ctx.Value("user_id"))
	}

	// Test invalid token
	_, err = authFunc(context.Background(), "invalid-token")
	if err != ErrUnauthorized {
		t.Errorf("MockAuthFuncMultiple with invalid token error = %v, want ErrUnauthorized", err)
	}
}

func TestMockAuthFuncAllowAll(t *testing.T) {
	authFunc := MockAuthFuncAllowAll()

	// Any token should work
	ctx, err := authFunc(context.Background(), "any-token")
	if err != nil {
		t.Errorf("MockAuthFuncAllowAll error = %v", err)
	}
	if ctx.Value("user_id") != "test-user" {
		t.Errorf("MockAuthFuncAllowAll user_id = %v, want test-user", ctx.Value("user_id"))
	}

	// Empty token should also work
	ctx, err = authFunc(context.Background(), "")
	if err != nil {
		t.Errorf("MockAuthFuncAllowAll with empty token error = %v", err)
	}
}

func TestTestServer_Close(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}

	// Get a connection before closing
	_ = ts.GRPCClientConn(context.Background())

	// Close should not panic
	ts.Close()

	// Double close should not panic
	ts.Close()
}

func TestTestServer_CustomHTTPHandler(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHTTPHandlerFunc("/custom", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"custom": true}`))
		}),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	resp, err := ts.HTTPClient().Get(ts.URL("/custom"))
	if err != nil {
		t.Fatalf("GET /custom error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /custom status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "custom") {
		t.Errorf("GET /custom body = %s, want to contain 'custom'", body)
	}
}

func TestTestServer_CORS(t *testing.T) {
	ts, err := NewTestServer(
		WithGRPCService(func(s grpc.ServiceRegistrar) {}),
		WithHealthCheck(),
		WithCORS(),
	)
	if err != nil {
		t.Fatalf("NewTestServer() error = %v", err)
	}
	defer ts.Close()

	// Make a request with Origin header
	req, _ := http.NewRequest("GET", ts.URL("/healthz"), nil)
	req.Header.Set("Origin", "http://example.com")

	resp, err := ts.HTTPClient().Do(req)
	if err != nil {
		t.Fatalf("GET /healthz with Origin error = %v", err)
	}
	defer resp.Body.Close()

	// Check CORS headers
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Error("Expected Access-Control-Allow-Origin header")
	}
}
