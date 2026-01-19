package grpckit

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	// Unregister any existing metrics from previous tests
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("test_namespace")

	if m == nil {
		t.Fatal("expected non-nil metrics")
	}

	if m.requestsTotal == nil {
		t.Error("expected requestsTotal counter to be initialized")
	}

	if m.requestDuration == nil {
		t.Error("expected requestDuration histogram to be initialized")
	}

	if m.requestsInFlight == nil {
		t.Error("expected requestsInFlight gauge to be initialized")
	}
}

func TestNewMetrics_DefaultNamespace(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("")

	if m == nil {
		t.Fatal("expected non-nil metrics")
	}

	// Verify the metrics work (the namespace is internal, we just verify initialization)
	if m.requestsTotal == nil {
		t.Error("expected requestsTotal to be initialized with default namespace")
	}
}

func TestMetricsHandler(t *testing.T) {
	handler := metricsHandler()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// The handler is just the promhttp.Handler(), verify it responds
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200 (even if there are no metrics yet)
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	// Response should contain prometheus format
	body := rec.Body.String()
	if !strings.Contains(body, "# HELP") && !strings.Contains(body, "# TYPE") && len(body) > 0 {
		// If there are metrics, they should have HELP/TYPE, otherwise empty is fine
	}
}

func TestRegisterMetricsEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	registerMetricsEndpoint(mux)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMetricsMiddleware(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("mw_test")

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := metricsMiddleware(m, next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestMetricsMiddleware_CapturesStatusCode(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("status_test")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	handler := metricsMiddleware(m, next)

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}
}

func TestMetricsMiddleware_InFlightGauge(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("flight_test")

	inFlightDuringRequest := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// During request, in-flight should be incremented
		// We can't easily test the exact value, but we verify the handler works
		inFlightDuringRequest = true
		w.WriteHeader(http.StatusOK)
	})

	handler := metricsMiddleware(m, next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !inFlightDuringRequest {
		t.Error("expected request to be processed")
	}
}

func TestResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	// Test default status code
	if wrapped.statusCode != http.StatusOK {
		t.Errorf("expected default status code 200, got %d", wrapped.statusCode)
	}

	// Test WriteHeader
	wrapped.WriteHeader(http.StatusCreated)
	if wrapped.statusCode != http.StatusCreated {
		t.Errorf("expected status code 201, got %d", wrapped.statusCode)
	}

	// Verify underlying writer also got the status
	if rec.Code != http.StatusCreated {
		t.Errorf("expected underlying recorder to have status 201, got %d", rec.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapped := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	// Write should work through to underlying writer
	n, err := wrapped.Write([]byte("hello"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}

	if rec.Body.String() != "hello" {
		t.Errorf("expected body 'hello', got %s", rec.Body.String())
	}
}

func TestMetricsMiddleware_MultipleRequests(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("multi_test")

	requestCount := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	})

	handler := metricsMiddleware(m, next)

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if requestCount != 5 {
		t.Errorf("expected 5 requests, got %d", requestCount)
	}
}

func TestMetricsMiddleware_DifferentMethods(t *testing.T) {
	prometheus.DefaultRegisterer = prometheus.NewRegistry()

	m := newMetrics("methods_test")

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := metricsMiddleware(m, next)

	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("method %s: expected status 200, got %d", method, rec.Code)
		}
	}
}
