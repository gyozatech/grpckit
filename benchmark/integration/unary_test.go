package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// BenchmarkUnaryCall_gRPC benchmarks simple gRPC unary calls.
func BenchmarkUnaryCall_gRPC(b *testing.B) {
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

	b.Run("Ping", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, &benchpb.PingRequest{})
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})

	b.Run("Echo", func(b *testing.B) {
		req := &benchpb.EchoRequest{
			Message: "Hello, World!",
			Data:    []byte("test data"),
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.Echo(ctx, req)
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
		}
	})

	b.Run("GetItem", func(b *testing.B) {
		req := &benchpb.GetItemRequest{Id: "test-item-123"}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.GetItem(ctx, req)
			if err != nil {
				b.Fatalf("GetItem failed: %v", err)
			}
		}
	})

	b.Run("CreateItem", func(b *testing.B) {
		req := &benchpb.CreateItemRequest{
			Name:        "Test Item",
			Description: "A test item for benchmarking",
			Price:       99.99,
			Tags:        []string{"test", "benchmark"},
		}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.CreateItem(ctx, req)
			if err != nil {
				b.Fatalf("CreateItem failed: %v", err)
			}
		}
	})

	b.Run("ListItems", func(b *testing.B) {
		req := &benchpb.ListItemsRequest{PageSize: 10}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := client.ListItems(ctx, req)
			if err != nil {
				b.Fatalf("ListItems failed: %v", err)
			}
		}
	})
}

// BenchmarkUnaryCall_REST benchmarks simple REST calls via grpc-gateway.
func BenchmarkUnaryCall_REST(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("Ping", func(b *testing.B) {
		url := baseURL + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("Echo", func(b *testing.B) {
		url := baseURL + "/api/v1/echo"
		body := `{"message":"Hello, World!","data":"dGVzdCBkYXRh","metadata":{"key1":"value1"}}`
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("GetItem", func(b *testing.B) {
		url := baseURL + "/api/v1/items/test-item-123"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("GetItem failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("CreateItem", func(b *testing.B) {
		url := baseURL + "/api/v1/items"
		body := `{"name":"Test Item","description":"A test item","price":99.99,"tags":["test"]}`
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("CreateItem failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("ListItems", func(b *testing.B) {
		url := baseURL + "/api/v1/items?page_size=10"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("ListItems failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkUnaryCall_Parallel benchmarks concurrent calls.
func BenchmarkUnaryCall_Parallel(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	b.Run("gRPC_Ping", func(b *testing.B) {
		client := h.GRPCClient()
		ctx := context.Background()
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

	b.Run("REST_Ping", func(b *testing.B) {
		client := h.HTTPClient()
		url := h.BaseURL() + "/api/v1/ping"

		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(url)
				if err != nil {
					b.Errorf("Ping failed: %v", err)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})
}

// BenchmarkPayloadSizes benchmarks different payload sizes.
func BenchmarkPayloadSizes(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	sizes := []struct {
		name string
		size int
	}{
		{"64B", 64},
		{"1KB", 1024},
		{"4KB", 4096},
		{"16KB", 16384},
	}

	client := h.GRPCClient()
	ctx := context.Background()

	for _, tc := range sizes {
		data := make([]byte, tc.size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		b.Run("gRPC_"+tc.name, func(b *testing.B) {
			req := &benchpb.EchoRequest{
				Message: "test",
				Data:    data,
			}
			b.ReportAllocs()
			b.SetBytes(int64(tc.size))
			for i := 0; i < b.N; i++ {
				_, err := client.Echo(ctx, req)
				if err != nil {
					b.Fatalf("Echo failed: %v", err)
				}
			}
		})
	}

	httpClient := h.HTTPClient()
	baseURL := h.BaseURL()

	for _, tc := range sizes {
		data := strings.Repeat("x", tc.size)

		b.Run("REST_"+tc.name, func(b *testing.B) {
			url := baseURL + "/api/v1/echo"
			body := `{"message":"` + data + `"}`
			b.ReportAllocs()
			b.SetBytes(int64(tc.size))
			for i := 0; i < b.N; i++ {
				resp, err := httpClient.Post(url, "application/json", strings.NewReader(body))
				if err != nil {
					b.Fatalf("Echo failed: %v", err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	}
}

// BenchmarkHTTPMethods benchmarks different HTTP methods.
func BenchmarkHTTPMethods(b *testing.B) {
	impl := service.New()

	h, err := harness.NewGRPCkitHarness(harness.GRPCkitConfig{
		ServiceImpl: impl,
	})
	if err != nil {
		b.Fatalf("failed to create harness: %v", err)
	}
	defer h.Close()

	client := h.HTTPClient()
	baseURL := h.BaseURL()

	b.Run("GET", func(b *testing.B) {
		url := baseURL + "/api/v1/ping"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("GET failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("POST", func(b *testing.B) {
		url := baseURL + "/api/v1/echo"
		body := `{"message":"test"}`
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Post(url, "application/json", strings.NewReader(body))
			if err != nil {
				b.Fatalf("POST failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("GET_WithPathParam", func(b *testing.B) {
		url := baseURL + "/api/v1/items/test-123"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("GET failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("GET_WithQueryParams", func(b *testing.B) {
		url := baseURL + "/api/v1/items?page_size=10&page_token=abc"
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(url)
			if err != nil {
				b.Fatalf("GET failed: %v", err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// Helper for creating HTTP requests with custom method
func doRequest(client *http.Client, method, url string, body string) (*http.Response, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return client.Do(req)
}
