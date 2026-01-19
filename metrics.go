package grpckit

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics for the server.
type Metrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

// newMetrics creates and registers Prometheus metrics.
func newMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "grpckit"
	}

	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_requests_in_flight",
				Help:      "Number of HTTP requests currently being processed",
			},
		),
	}

	// Register metrics
	prometheus.MustRegister(m.requestsTotal)
	prometheus.MustRegister(m.requestDuration)
	prometheus.MustRegister(m.requestsInFlight)

	return m
}

// metricsHandler returns the Prometheus metrics endpoint handler.
func metricsHandler() http.Handler {
	return promhttp.Handler()
}

// registerMetricsEndpoint registers the /metrics endpoint on the mux.
func registerMetricsEndpoint(mux *http.ServeMux) {
	mux.Handle("/metrics", metricsHandler())
}

// metricsMiddleware wraps an HTTP handler to collect metrics.
func metricsMiddleware(m *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()
		statusStr := http.StatusText(wrapped.statusCode)

		// Normalize path to prevent cardinality explosion from dynamic IDs
		normalizedPath := normalizePath(r.URL.Path)

		m.requestsTotal.WithLabelValues(r.Method, normalizedPath, statusStr).Inc()
		m.requestDuration.WithLabelValues(r.Method, normalizedPath).Observe(duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// normalizePath normalizes URL paths for metrics labels to prevent cardinality explosion.
// It replaces dynamic path segments (IDs, UUIDs) with placeholders.
// Examples:
//   - /api/users/123 -> /api/users/:id
//   - /api/items/550e8400-e29b-41d4-a716-446655440000 -> /api/items/:id
//   - /healthz -> /healthz (unchanged)
func normalizePath(path string) string {
	if path == "" || path == "/" {
		return path
	}

	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part != "" && isLikelyID(part) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

// isLikelyID checks if a path segment looks like a dynamic ID.
// Matches numeric IDs, UUIDs, and other common ID patterns.
func isLikelyID(s string) bool {
	if s == "" {
		return false
	}

	// Numeric IDs (any length)
	if _, err := strconv.ParseInt(s, 10, 64); err == nil {
		return true
	}

	// UUID-like (36 chars with 4 dashes: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}

	// Short UUIDs or base64-like IDs (20+ chars with alphanumeric and common ID chars)
	if len(s) >= 20 && isAlphanumericWithIDChars(s) {
		return true
	}

	return false
}

// isAlphanumericWithIDChars checks if string contains only alphanumeric chars plus - and _.
func isAlphanumericWithIDChars(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}
