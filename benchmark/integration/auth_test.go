package integration

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
	"google.golang.org/grpc/metadata"
)

// BenchmarkAuth_REST benchmarks REST calls with authentication.
func BenchmarkAuth_REST(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
		EnableAuth:  true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("WithToken", func(b *testing.B) {
		url := baseURL + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("WithoutToken", func(b *testing.B) {
		url := baseURL + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			// Expect 401, but we're measuring overhead not correctness
		}
	})
}

// BenchmarkAuth_gRPC benchmarks gRPC calls with authentication.
func BenchmarkAuth_gRPC(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
		EnableAuth:  true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.GRPCClient()

	b.Run("WithToken", func(b *testing.B) {
		md := metadata.Pairs("authorization", "Bearer test-token")
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		req := &benchpb.PingRequest{}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})

	b.Run("WithoutToken", func(b *testing.B) {
		ctx := context.Background()
		req := &benchpb.PingRequest{}

		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// This will fail auth but we're measuring overhead
			_, _ = client.Ping(ctx, req)
		}
	})
}

// BenchmarkAuth_Comparison compares auth vs no-auth overhead.
func BenchmarkAuth_Comparison(b *testing.B) {
	impl := service.New()

	// Create harness without auth
	hNoAuth, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hNoAuth.Close()

	// Create harness with auth
	hAuth, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
		EnableAuth:  true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer hAuth.Close()

	b.Run("REST_NoAuth", func(b *testing.B) {
		client := hNoAuth.HTTPClient()
		url := hNoAuth.BaseURL() + "/api/v1/ping"
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

	b.Run("REST_WithAuth", func(b *testing.B) {
		client := hAuth.HTTPClient()
		url := hAuth.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := client.Do(req)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("gRPC_NoAuth", func(b *testing.B) {
		client := hNoAuth.GRPCClient()
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

	b.Run("gRPC_WithAuth", func(b *testing.B) {
		client := hAuth.GRPCClient()
		md := metadata.Pairs("authorization", "Bearer test-token")
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		req := &benchpb.PingRequest{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})
}

// BenchmarkAuth_Parallel benchmarks auth under concurrent load.
func BenchmarkAuth_Parallel(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
		EnableAuth:  true,
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

	b.Run("gRPC", func(b *testing.B) {
		client := h.GRPCClient()
		md := metadata.Pairs("authorization", "Bearer test-token")
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		req := &benchpb.PingRequest{}
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := client.Ping(ctx, req)
				if err != nil {
					b.Errorf("Ping failed: %v", err)
				}
			}
		})
	})
}
