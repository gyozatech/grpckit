package grpckit

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSwaggerHandler(t *testing.T) {
	// Create temp swagger file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "swagger.json")

	validSpec := `{"openapi": "3.0.0", "info": {"title": "Test API", "version": "1.0"}}`
	if err := os.WriteFile(specPath, []byte(validSpec), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	handler, err := newSwaggerHandler(specPath)
	if err != nil {
		t.Fatalf("newSwaggerHandler failed: %v", err)
	}

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	if handler.specPath != specPath {
		t.Errorf("expected specPath %s, got %s", specPath, handler.specPath)
	}

	if string(handler.specData) != validSpec {
		t.Errorf("expected specData %s, got %s", validSpec, string(handler.specData))
	}
}

func TestNewSwaggerHandler_FileNotFound(t *testing.T) {
	_, err := newSwaggerHandler("/nonexistent/swagger.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestNewSwaggerHandler_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "swagger.json")

	if err := os.WriteFile(specPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	_, err := newSwaggerHandler(specPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestNewSwaggerHandlerFromBytes(t *testing.T) {
	validSpec := []byte(`{"openapi": "3.0.0"}`)

	handler, err := newSwaggerHandlerFromBytes(validSpec)
	if err != nil {
		t.Fatalf("newSwaggerHandlerFromBytes failed: %v", err)
	}

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	if string(handler.specData) != string(validSpec) {
		t.Errorf("expected specData to match input")
	}

	// specPath should be empty for bytes-based handler
	if handler.specPath != "" {
		t.Errorf("expected empty specPath, got %s", handler.specPath)
	}
}

func TestNewSwaggerHandlerFromBytes_InvalidJSON(t *testing.T) {
	_, err := newSwaggerHandlerFromBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSwaggerHandler_UIHandler(t *testing.T) {
	handler, _ := newSwaggerHandlerFromBytes([]byte(`{"openapi": "3.0.0"}`))
	uiHandler := handler.UIHandler()

	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()

	uiHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Errorf("expected Content-Type text/html, got %s", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Error("expected swagger-ui in HTML body")
	}

	// Template escapes slashes as \/ for JS safety
	if !strings.Contains(body, "/swagger/spec.json") && !strings.Contains(body, `\/swagger\/spec.json`) {
		t.Error("expected spec URL in HTML body")
	}
}

func TestSwaggerHandler_SpecHandler(t *testing.T) {
	specData := []byte(`{"openapi": "3.0.0", "info": {"title": "Test"}}`)
	handler, _ := newSwaggerHandlerFromBytes(specData)
	specHandler := handler.SpecHandler()

	req := httptest.NewRequest(http.MethodGet, "/swagger/spec.json", nil)
	rec := httptest.NewRecorder()

	specHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	if rec.Body.String() != string(specData) {
		t.Errorf("expected spec data in response body")
	}
}

func TestRegisterSwaggerEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "swagger.json")

	if err := os.WriteFile(specPath, []byte(`{"openapi": "3.0.0"}`), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	mux := http.NewServeMux()
	err := registerSwaggerEndpoints(mux, specPath)
	if err != nil {
		t.Fatalf("registerSwaggerEndpoints failed: %v", err)
	}

	// Test /swagger/ (UI)
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/swagger/ expected status 200, got %d", rec.Code)
	}

	// Test /swagger/spec.json
	req = httptest.NewRequest(http.MethodGet, "/swagger/spec.json", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/swagger/spec.json expected status 200, got %d", rec.Code)
	}
}

func TestRegisterSwaggerEndpoints_InvalidFile(t *testing.T) {
	mux := http.NewServeMux()
	err := registerSwaggerEndpoints(mux, "/nonexistent/swagger.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestRegisterSwaggerEndpointsFromBytes(t *testing.T) {
	mux := http.NewServeMux()
	specData := []byte(`{"openapi": "3.0.0"}`)

	err := registerSwaggerEndpointsFromBytes(mux, specData)
	if err != nil {
		t.Fatalf("registerSwaggerEndpointsFromBytes failed: %v", err)
	}

	// Test /swagger/
	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/swagger/ expected status 200, got %d", rec.Code)
	}
}

func TestRegisterSwaggerEndpointsFromBytes_InvalidJSON(t *testing.T) {
	mux := http.NewServeMux()
	err := registerSwaggerEndpointsFromBytes(mux, []byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRegisterSwaggerHandler_Routing(t *testing.T) {
	handler, _ := newSwaggerHandlerFromBytes([]byte(`{"openapi": "3.0.0"}`))
	mux := http.NewServeMux()
	registerSwaggerHandler(mux, handler)

	tests := []struct {
		path           string
		expectedStatus int
		description    string
	}{
		{"/swagger/", http.StatusOK, "UI at /swagger/"},
		{"/swagger", http.StatusMovedPermanently, "UI at /swagger (redirects to /swagger/)"},
		{"/swagger/spec.json", http.StatusOK, "Spec at /swagger/spec.json"},
		{"/swagger/unknown", http.StatusNotFound, "404 for unknown paths"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.path, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestRegisterSwaggerNotFound(t *testing.T) {
	mux := http.NewServeMux()
	registerSwaggerNotFound(mux)

	req := httptest.NewRequest(http.MethodGet, "/swagger/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "make swagger") {
		t.Error("expected helpful message about running 'make swagger'")
	}
}

func TestSwaggerUIHTML_Template(t *testing.T) {
	// Verify the template contains expected elements
	if !strings.Contains(swaggerUIHTML, "swagger-ui") {
		t.Error("template should contain swagger-ui")
	}

	if !strings.Contains(swaggerUIHTML, "{{.SpecURL}}") {
		t.Error("template should contain SpecURL placeholder")
	}

	if !strings.Contains(swaggerUIHTML, "SwaggerUIBundle") {
		t.Error("template should contain SwaggerUIBundle")
	}
}

func TestSetSwaggerData(t *testing.T) {
	// Save original
	original := globalSwaggerData
	defer func() {
		globalSwaggerData = original
	}()

	testData := []byte(`{"test": "data"}`)
	SetSwaggerData(testData)

	if string(globalSwaggerData) != string(testData) {
		t.Errorf("expected globalSwaggerData to be set")
	}
}
