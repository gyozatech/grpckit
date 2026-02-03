package integration

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// BenchmarkMiddleware_Metrics benchmarks the metrics middleware overhead.
func BenchmarkMiddleware_Metrics(b *testing.B) {
	impl := service.New()

	// Create harness without metrics
	hNoMetrics, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hNoMetrics.Close()

	// Create harness with metrics
	hMetrics, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:   impl,
		EnableMetrics: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hMetrics.Close()

	b.Run("REST_NoMetrics", func(b *testing.B) {
		client := hNoMetrics.HTTPClient()
		url := hNoMetrics.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("REST_WithMetrics", func(b *testing.B) {
		client := hMetrics.HTTPClient()
		url := hMetrics.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkMiddleware_CORS benchmarks the CORS middleware overhead.
func BenchmarkMiddleware_CORS(b *testing.B) {
	impl := service.New()

	// Create harness without CORS
	hNoCORS, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hNoCORS.Close()

	// Create harness with CORS
	hCORS, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
		EnableCORS:  true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hCORS.Close()

	b.Run("REST_NoCORS", func(b *testing.B) {
		client := hNoCORS.HTTPClient()
		url := hNoCORS.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("REST_WithCORS", func(b *testing.B) {
		client := hCORS.HTTPClient()
		url := hCORS.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("Preflight_WithCORS", func(b *testing.B) {
		client := hCORS.HTTPClient()
		url := hCORS.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("OPTIONS", url, nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")
			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkMiddleware_FullStack benchmarks all middleware combined.
func BenchmarkMiddleware_FullStack(b *testing.B) {
	impl := service.New()

	// Create harness with no middleware
	hMinimal, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hMinimal.Close()

	// Create harness with all middleware
	hFull, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:   impl,
		EnableAuth:    true,
		EnableMetrics: true,
		EnableCORS:    true,
		EnableHealth:  true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hFull.Close()

	b.Run("REST_Minimal", func(b *testing.B) {
		client := hMinimal.HTTPClient()
		url := hMinimal.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("REST_FullStack", func(b *testing.B) {
		client := hFull.HTTPClient()
		url := hFull.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Origin", "https://example.com")
			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("gRPC_Minimal", func(b *testing.B) {
		client := hMinimal.GRPCClient()
		ctx := context.Background()
		req := &benchpb.PingRequest{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})

	b.Run("gRPC_FullStack", func(b *testing.B) {
		client := hFull.GRPCClient()
		ctx := context.Background()
		req := &benchpb.PingRequest{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// Note: gRPC doesn't go through CORS or HTTP metrics middleware
			_, err := client.Ping(ctx, req)
			if err != nil {
				// Auth will fail without token, but we're measuring overhead
			}
		}
	})
}

// BenchmarkMiddleware_Parallel benchmarks middleware under concurrent load.
func BenchmarkMiddleware_Parallel(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:   impl,
		EnableAuth:    true,
		EnableMetrics: true,
		EnableCORS:    true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	b.Run("REST", func(b *testing.B) {
		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("GET", url, nil)
				req.Header.Set("Authorization", "Bearer test-token")
				resp, err := client.Do(req)
				if err != nil {
					b.Errorf("request failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})
}
