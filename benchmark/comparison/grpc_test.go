package comparison

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/gyozatech/grpckit/benchmark/internal/harness"
	"github.com/gyozatech/grpckit/benchmark/internal/service"
	benchpb "github.com/gyozatech/grpckit/benchmark/proto/gen"
)

// BenchmarkGRPCUnary_Ping compares gRPC unary call performance across frameworks.
func BenchmarkGRPCUnary_Ping(b *testing.B) {
	impl := service.New()
	ctx := context.Background()
	req := &benchpb.PingRequest{}

	b.Run("gRPCkit", func(b *testing.B) {
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
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
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

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
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
		connectReq := connect.NewRequest(&benchpb.PingRequest{})
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, connectReq)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
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

		client := harness.NewTwirpClient(h.HTTPClient(), h.BaseURL())
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
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

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Ping(ctx, req)
			if err != nil {
				b.Fatalf("Ping failed: %v", err)
			}
		}
	})
}

// BenchmarkGRPCUnary_Echo compares gRPC Echo call performance.
func BenchmarkGRPCUnary_Echo(b *testing.B) {
	impl := service.New()
	ctx := context.Background()
	req := &benchpb.EchoRequest{
		Message: "Hello, World!",
		Data:    []byte("test data"),
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.Run("gRPCkit", func(b *testing.B) {
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
		for i := 0; i < b.N; i++ {
			_, err := client.Echo(ctx, req)
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
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

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Echo(ctx, req)
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
		}
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
		connectReq := connect.NewRequest(req)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Echo(ctx, connectReq)
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
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

		client := harness.NewTwirpClient(h.HTTPClient(), h.BaseURL())
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.Echo(ctx, req)
			if err != nil {
				b.Fatalf("Echo failed: %v", err)
			}
		}
	})
}

// BenchmarkGRPCUnary_Parallel compares parallel gRPC performance.
func BenchmarkGRPCUnary_Parallel(b *testing.B) {
	impl := service.New()
	ctx := context.Background()
	req := &benchpb.PingRequest{}

	b.Run("gRPCkit", func(b *testing.B) {
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
				_, err := client.Ping(ctx, req)
				if err != nil {
					b.Errorf("Ping failed: %v", err)
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

		client := h.GRPCClient()
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, err := client.Ping(ctx, req)
				if err != nil {
					b.Errorf("Ping failed: %v", err)
				}
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
				_, err := client.Ping(ctx, connectReq)
				if err != nil {
					b.Errorf("Ping failed: %v", err)
				}
			}
		})
	})
}
