package comparison

import (
	"io"
	"strings"
	"testing"

	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
)

// BenchmarkREST_Ping compares REST endpoint performance across frameworks.
func BenchmarkREST_Ping(b *testing.B) {
	impl := service.New()

	b.Run("gRPCkit", func(b *testing.B) {
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
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	// Twirp uses POST for all calls, so it's not directly comparable for GET endpoints
	b.Run("Twirp_POST", func(b *testing.B) {
		h, err := harness.NewTwirpHarness(harness.TwirpConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/twirp/benchmark.v1.BenchmarkService/Ping"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader("{}"))
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkREST_Echo compares REST POST performance.
func BenchmarkREST_Echo(b *testing.B) {
	impl := service.New()
	body := `{"message":"Hello, World!","metadata":{"key1":"value1"}}`

	b.Run("gRPCkit", func(b *testing.B) {
		h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/echo"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		url := h.BaseURL() + "/api/v1/echo"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		url := h.BaseURL() + "/api/v1/echo"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("Twirp", func(b *testing.B) {
		h, err := harness.NewTwirpHarness(harness.TwirpConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/twirp/benchmark.v1.BenchmarkService/Echo"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkREST_GetItem compares path parameter extraction performance.
func BenchmarkREST_GetItem(b *testing.B) {
	impl := service.New()

	b.Run("gRPCkit", func(b *testing.B) {
		h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
			ServiceImpl: impl,
		})
		if err != nil {
			b.Fatalf("failed to create harness: %v", err)
		}
		defer h.Close()

		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/items/test-item-123"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		url := h.BaseURL() + "/api/v1/items/test-item-123"
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("request failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
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
		url := h.BaseURL() + "/api/v1/items/test-item-123"
		b.ReportAllocs()
		b.ResetTimer()
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

// BenchmarkREST_Parallel compares parallel REST performance.
func BenchmarkREST_Parallel(b *testing.B) {
	impl := service.New()

	b.Run("gRPCkit", func(b *testing.B) {
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
				if err != nil {
					b.Errorf("request failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
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
				if err != nil {
					b.Errorf("request failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
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
