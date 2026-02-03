package harness

import (
	"errors"
	"net/http"

	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// Common errors.
var (
	ErrNilServiceImpl = errors.New("service implementation cannot be nil")
)

// Harness defines the common interface for all test harnesses.
type Harness interface {
	// GRPCClient returns the gRPC client (may be nil if not supported).
	GRPCClient() benchpb.BenchmarkServiceClient
	// HTTPClient returns the HTTP client.
	HTTPClient() *http.Client
	// BaseURL returns the base URL for REST requests.
	BaseURL() string
	// Close shuts down the harness.
	Close()
}

// Ensure all harnesses implement the interface.
var (
	_ Harness = (*GRPCkitHarness)(nil)
	_ Harness = (*VanillaHarness)(nil)
	_ Harness = (*ConnectHarness)(nil)
	_ Harness = (*TwirpHarness)(nil)
	_ Harness = (*GinHarness)(nil)
)
