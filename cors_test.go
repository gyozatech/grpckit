package grpckit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Errorf("expected AllowedOrigins [*], got %v", cfg.AllowedOrigins)
	}

	expectedMethods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
	}
	if len(cfg.AllowedMethods) != len(expectedMethods) {
		t.Errorf("expected %d allowed methods, got %d", len(expectedMethods), len(cfg.AllowedMethods))
	}

	if !cfg.AllowCredentials {
		t.Error("expected AllowCredentials to be true")
	}

	if cfg.MaxAge != 86400 {
		t.Errorf("expected MaxAge 86400, got %d", cfg.MaxAge)
	}
}

func TestCORSMiddleware_Wildcard(t *testing.T) {
	cfg := DefaultCORSConfig()
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Vary") != "Origin" {
		t.Errorf("expected Vary Origin, got %s", rec.Header().Get("Vary"))
	}
}

func TestCORSMiddleware_SpecificOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
		AllowCredentials: true,
	}
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test allowed origin
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin https://example.com, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials true, got %s", rec.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
	}
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no Access-Control-Allow-Origin for disallowed origin, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	cfg := DefaultCORSConfig()
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("should not reach here"))
	}))

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 for preflight, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Max-Age") == "" {
		t.Error("expected Access-Control-Max-Age header for preflight")
	}

	if rec.Body.String() != "" {
		t.Errorf("expected empty body for preflight, got %s", rec.Body.String())
	}
}

func TestCORSMiddleware_NoOrigin(t *testing.T) {
	cfg := DefaultCORSConfig()
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Origin header
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// No CORS headers should be set for same-origin requests
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS headers without Origin, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins: []string{"*"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Request-Id"},
	}
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	exposedHeaders := rec.Header().Get("Access-Control-Expose-Headers")
	if exposedHeaders == "" {
		t.Error("expected Access-Control-Expose-Headers header")
	}
}

func TestCORSMiddleware_CredentialsNotSetForWildcard(t *testing.T) {
	cfg := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true, // This should be ignored for wildcard
	}
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Credentials should not be set for wildcard origin
	if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Error("Access-Control-Allow-Credentials should not be set for wildcard origin")
	}
}

func TestCORSMiddleware_DefaultsApplied(t *testing.T) {
	// Empty config should get defaults
	cfg := CORSConfig{}
	middleware := corsMiddleware(cfg)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should have wildcard origin
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected default wildcard origin, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// Should have default methods
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected default methods to be set")
	}
}
