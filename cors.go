package grpckit

import (
	"net/http"
	"strings"
)

// CORSConfig configures CORS (Cross-Origin Resource Sharing) behavior.
type CORSConfig struct {
	// AllowedOrigins is a list of origins that are allowed to make requests.
	// Use "*" to allow all origins (default if empty).
	AllowedOrigins []string

	// AllowedMethods is a list of HTTP methods allowed for CORS requests.
	// Default: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
	AllowedMethods []string

	// AllowedHeaders is a list of headers that are allowed in requests.
	// Default: Accept, Content-Type, Content-Length, Accept-Encoding,
	// Authorization, X-Requested-With, X-CSRF-Token
	AllowedHeaders []string

	// ExposedHeaders is a list of headers that browsers are allowed to access.
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials (cookies, authorization headers)
	// are allowed in CORS requests. Default: true
	AllowCredentials bool

	// MaxAge is the max duration (in seconds) that preflight results can be cached.
	// Default: 86400 (24 hours)
	MaxAge int
}

// DefaultCORSConfig returns a permissive CORS configuration that allows
// requests from any origin. Suitable for development and public APIs.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Accept",
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"Authorization",
			"X-Requested-With",
			"X-CSRF-Token",
			"X-Request-ID",
			"Origin",
		},
		ExposedHeaders:   []string{},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// corsMiddleware creates an HTTP middleware that handles CORS.
func corsMiddleware(cfg CORSConfig) HTTPMiddleware {
	// Apply defaults for empty fields
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = []string{"*"}
	}
	if len(cfg.AllowedMethods) == 0 {
		cfg.AllowedMethods = DefaultCORSConfig().AllowedMethods
	}
	if len(cfg.AllowedHeaders) == 0 {
		cfg.AllowedHeaders = DefaultCORSConfig().AllowedHeaders
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 86400
	}

	// Pre-compute header values
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := http.StatusOK // Will be converted to string in handler

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Determine the allowed origin to return
			allowedOrigin := ""
			if origin != "" {
				allowedOrigin = getAllowedOrigin(origin, cfg.AllowedOrigins)
			}

			// Set CORS headers if origin is allowed
			if allowedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)

				if len(cfg.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", exposedHeaders)
				}

				if cfg.AllowCredentials && allowedOrigin != "*" {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Vary header tells caches that response varies based on Origin
				w.Header().Add("Vary", "Origin")
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Max-Age", intToString(cfg.MaxAge))
				w.WriteHeader(maxAge)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getAllowedOrigin checks if the origin is in the allowed list and returns
// the appropriate value for the Access-Control-Allow-Origin header.
func getAllowedOrigin(origin string, allowedOrigins []string) string {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return "*"
		}
		if allowed == origin {
			return origin
		}
	}
	return ""
}

// intToString converts an int to string without importing strconv.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	// Handle the conversion manually
	negative := n < 0
	if negative {
		n = -n
	}

	// Build digits in reverse
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}

	// Reverse the digits
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
