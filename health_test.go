package grpckit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHealthHandler(t *testing.T) {
	h := newHealthHandler()

	if h == nil {
		t.Fatal("expected non-nil health handler")
	}

	// Should start ready by default
	if !h.IsReady() {
		t.Error("expected health handler to start ready")
	}
}

func TestHealthHandler_SetReady(t *testing.T) {
	h := newHealthHandler()

	// Default is ready
	if !h.IsReady() {
		t.Error("expected default ready state to be true")
	}

	// Set not ready
	h.SetReady(false)
	if h.IsReady() {
		t.Error("expected ready state to be false after SetReady(false)")
	}

	// Set ready again
	h.SetReady(true)
	if !h.IsReady() {
		t.Error("expected ready state to be true after SetReady(true)")
	}
}

func TestHealthHandler_LivenessHandler(t *testing.T) {
	h := newHealthHandler()
	handler := h.LivenessHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", rec.Header().Get("Content-Type"))
	}

	var status HealthStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if status.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", status.Status)
	}
}

func TestHealthHandler_LivenessHandler_AlwaysReturns200(t *testing.T) {
	h := newHealthHandler()
	handler := h.LivenessHandler()

	// Even when not ready, liveness should return 200
	h.SetReady(false)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("liveness should always return 200, got %d", rec.Code)
	}
}

func TestHealthHandler_ReadinessHandler_Ready(t *testing.T) {
	h := newHealthHandler()
	handler := h.ReadinessHandler()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 when ready, got %d", rec.Code)
	}

	var status HealthStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if status.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", status.Status)
	}
}

func TestHealthHandler_ReadinessHandler_NotReady(t *testing.T) {
	h := newHealthHandler()
	h.SetReady(false)
	handler := h.ReadinessHandler()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 when not ready, got %d", rec.Code)
	}

	var status HealthStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if status.Status != "not ready" {
		t.Errorf("expected status 'not ready', got %q", status.Status)
	}
}

func TestRegisterHealthEndpoints(t *testing.T) {
	h := newHealthHandler()
	mux := http.NewServeMux()

	registerHealthEndpoints(mux, h)

	// Test /healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz expected status 200, got %d", rec.Code)
	}

	// Test /readyz
	req = httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz expected status 200, got %d", rec.Code)
	}
}

func TestHealthStatus_JSONSerialization(t *testing.T) {
	status := HealthStatus{Status: "ok"}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	expected := `{"status":"ok"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}

	var unmarshaled HealthStatus
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if unmarshaled.Status != status.Status {
		t.Errorf("expected status %q, got %q", status.Status, unmarshaled.Status)
	}
}

func TestHealthHandler_ConcurrentAccess(t *testing.T) {
	h := newHealthHandler()

	done := make(chan bool)

	// Concurrent writers
	for i := 0; i < 10; i++ {
		go func(ready bool) {
			for j := 0; j < 100; j++ {
				h.SetReady(ready)
			}
			done <- true
		}(i%2 == 0)
	}

	// Concurrent readers
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = h.IsReady()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}
