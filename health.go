package grpckit

import (
	"net/http"
	"sync/atomic"
)

// Pre-computed response bytes to avoid JSON encoding on every request.
var (
	healthOKResponse       = []byte(`{"status":"ok"}`)
	healthNotReadyResponse = []byte(`{"status":"not ready"}`)
)

// HealthStatus represents the health check response.
type HealthStatus struct {
	Status string `json:"status"`
}

// healthHandler manages health check state and handlers.
type healthHandler struct {
	ready atomic.Bool
}

// newHealthHandler creates a new health handler.
func newHealthHandler() *healthHandler {
	h := &healthHandler{}
	h.ready.Store(true) // Start ready by default
	return h
}

// SetReady sets the readiness state.
func (h *healthHandler) SetReady(ready bool) {
	h.ready.Store(ready)
}

// IsReady returns the current readiness state.
func (h *healthHandler) IsReady() bool {
	return h.ready.Load()
}

// LivenessHandler returns the liveness probe handler.
// This endpoint always returns 200 OK if the server is running.
// Uses pre-computed response bytes for optimal performance.
func (h *healthHandler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(healthOKResponse)
	}
}

// ReadinessHandler returns the readiness probe handler.
// This endpoint returns 200 OK if the server is ready to accept traffic.
// Uses pre-computed response bytes for optimal performance.
func (h *healthHandler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if h.IsReady() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(healthOKResponse)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write(healthNotReadyResponse)
		}
	}
}

// registerHealthEndpoints registers health check endpoints on the mux.
func registerHealthEndpoints(mux *http.ServeMux, h *healthHandler) {
	mux.HandleFunc("/healthz", h.LivenessHandler())
	mux.HandleFunc("/readyz", h.ReadinessHandler())
}
