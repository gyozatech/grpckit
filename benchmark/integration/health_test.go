package integration

import (
	"io"
	"testing"

	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
)

// BenchmarkHealth_Endpoints benchmarks health check endpoint throughput.
func BenchmarkHealth_Endpoints(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:  impl,
		EnableHealth: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("Healthz", func(b *testing.B) {
		url := baseURL + "/healthz"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("health check failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("Livez", func(b *testing.B) {
		url := baseURL + "/livez"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("liveness check failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("Readyz", func(b *testing.B) {
		url := baseURL + "/readyz"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("readiness check failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkHealth_Parallel benchmarks health endpoints under concurrent load.
func BenchmarkHealth_Parallel(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:  impl,
		EnableHealth: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("Healthz", func(b *testing.B) {
		url := baseURL + "/healthz"
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err != nil {
					b.Errorf("health check failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})

	b.Run("Livez", func(b *testing.B) {
		url := baseURL + "/livez"
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err != nil {
					b.Errorf("liveness check failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})
}

// BenchmarkHealth_HighFrequency simulates Kubernetes-style frequent probing.
func BenchmarkHealth_HighFrequency(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:  impl,
		EnableHealth: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	// Simulate typical K8s probe patterns
	urls := []string{
		baseURL + "/healthz",
		baseURL + "/livez",
		baseURL + "/readyz",
	}

	b.Run("MixedProbes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			url := urls[i%len(urls)]
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("probe failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("MixedProbes_Parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				url := urls[i%len(urls)]
				resp, err := client.Get(url)
				if err != nil {
					b.Errorf("probe failed: %v", err)
					i++
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				i++
			}
		})
	})
}

// BenchmarkHealth_vs_API compares health endpoint performance to API endpoints.
func BenchmarkHealth_vs_API(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl:   impl,
		EnableHealth:  true,
		EnableMetrics: true,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("Health_Endpoint", func(b *testing.B) {
		url := baseURL + "/healthz"
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

	b.Run("API_Endpoint", func(b *testing.B) {
		url := baseURL + "/api/v1/ping"
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
