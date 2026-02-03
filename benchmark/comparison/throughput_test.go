package comparison

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// BenchmarkThroughput_GRPC measures maximum gRPC throughput under load.
func BenchmarkThroughput_GRPC(b *testing.B) {
	impl := service.New()
	ctx := context.Background()
	req := &benchpb.PingRequest{}

	b.Run("GRPCkit", func(b *testing.B) {
		h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = client.Ping(ctx, req)
			}
		})
	})

	b.Run("Vanilla", func(b *testing.B) {
		h, err := harness.NewVanillaHarness(harness.VanillaConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = client.Ping(ctx, req)
			}
		})
	})

	b.Run("Connect", func(b *testing.B) {
		h, err := harness.NewConnectHarness(harness.ConnectConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.ConnectClient()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Create new request per iteration to avoid concurrent map access
				connectReq := connect.NewRequest(&benchpb.PingRequest{})
				_, _ = client.Ping(ctx, connectReq)
			}
		})
	})
}

// BenchmarkThroughput_REST measures maximum REST throughput under load.
func BenchmarkThroughput_REST(b *testing.B) {
	impl := service.New()

	b.Run("GRPCkit", func(b *testing.B) {
		h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		})
	})

	b.Run("Vanilla", func(b *testing.B) {
		h, err := harness.NewVanillaHarness(harness.VanillaConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		})
	})

	b.Run("Gin", func(b *testing.B) {
		h, err := harness.NewGinHarness(harness.GinConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/ping"
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		})
	})
}

// BenchmarkThroughput_Mixed simulates realistic mixed workload.
func BenchmarkThroughput_Mixed(b *testing.B) {
	impl := service.New()

	b.Run("GRPCkit", func(b *testing.B) {
		h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
			ServiceImpl:   impl,
			EnableMetrics: true,
			EnableHealth:  true,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		grpcClient := h.GRPCClient()
		baseURL := h.BaseURL()
		ctx := context.Background()

		urls := []string{
			baseURL + "/api/v1/ping",
			baseURL + "/api/v1/items/123",
			baseURL + "/healthz",
		}

		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				switch i % 4 {
				case 0: // REST ping
					resp, _ := client.Get(urls[0])
					if resp != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				case 1: // REST get item
					resp, _ := client.Get(urls[1])
					if resp != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				case 2: // gRPC ping
					grpcClient.Ping(ctx, &benchpb.PingRequest{})
				case 3: // Health check
					resp, _ := client.Get(urls[2])
					if resp != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
				}
				i++
			}
		})
	})
}

// BenchmarkLatencyPercentiles measures latency distribution.
// This is a special benchmark that reports percentile data.
func BenchmarkLatencyPercentiles(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping latency percentile benchmark in short mode")
	}

	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.GRPCClient()
	ctx := context.Background()
	req := &benchpb.PingRequest{}

	// Collect latency samples
	const numSamples = 10000
	latencies := make([]time.Duration, numSamples)

	for i := 0; i < numSamples; i++ {
		start := time.Now()
		_, _ = client.Ping(ctx, req)
		latencies[i] = time.Since(start)
	}

	// Sort and report percentiles
	var total time.Duration
	for _, l := range latencies {
		total += l
	}

	b.ReportMetric(float64(total.Nanoseconds())/float64(numSamples), "ns/op")
	b.ReportMetric(float64(latencies[numSamples/2].Nanoseconds()), "p50-ns")
	b.ReportMetric(float64(latencies[numSamples*95/100].Nanoseconds()), "p95-ns")
	b.ReportMetric(float64(latencies[numSamples*99/100].Nanoseconds()), "p99-ns")
}

// BenchmarkSustainedLoad measures performance under sustained load.
func BenchmarkSustainedLoad(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping sustained load benchmark in short mode")
	}

	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:   impl,
		EnableMetrics: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	url := h.BaseURL() + "/api/v1/ping"

	// Run sustained load for the duration of the benchmark
	var ops atomic.Int64
	var errors atomic.Int64

	b.ReportAllocs()
	b.ResetTimer()

	const numWorkers = 10
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < b.N/numWorkers; i++ {
				resp, err := client.Get(url)
				if err != nil {
					errors.Add(1)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				ops.Add(1)
			}
		}()
	}
	wg.Wait()

	b.ReportMetric(float64(errors.Load()), "errors")
	b.ReportMetric(float64(ops.Load()), "successful-ops")
}
