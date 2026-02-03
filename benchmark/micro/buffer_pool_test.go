package micro

import (
	"bytes"
	"sync"
	"testing"

	"github.com/gyozatech/grpckit"
)

// BenchmarkBufferPool benchmarks the buffer pool get/put operations.
func BenchmarkBufferPool(b *testing.B) {
	b.Run("GetPut", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := grpckit.GetBuffer()
			buf.WriteString("test data")
			grpckit.PutBuffer(buf)
		}
	})

	b.Run("GetPut_LargeWrite", func(b *testing.B) {
		data := bytes.Repeat([]byte("x"), 4096)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := grpckit.GetBuffer()
			buf.Write(data)
			grpckit.PutBuffer(buf)
		}
	})

	b.Run("NoPool", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.WriteString("test data")
			_ = buf
		}
	})

	b.Run("NoPool_LargeWrite", func(b *testing.B) {
		data := bytes.Repeat([]byte("x"), 4096)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Write(data)
			_ = buf
		}
	})
}

// BenchmarkBufferPool_Parallel benchmarks buffer pool under concurrent load.
func BenchmarkBufferPool_Parallel(b *testing.B) {
	b.Run("Pooled", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := grpckit.GetBuffer()
				buf.WriteString("test data for parallel benchmark")
				grpckit.PutBuffer(buf)
			}
		})
	})

	b.Run("NotPooled", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				buf := new(bytes.Buffer)
				buf.WriteString("test data for parallel benchmark")
				_ = buf
			}
		})
	})
}

// BenchmarkBufferPool_HitRate measures pool hit rate under different patterns.
func BenchmarkBufferPool_HitRate(b *testing.B) {
	// Sequential access - should have high hit rate
	b.Run("Sequential", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := grpckit.GetBuffer()
			buf.WriteString("data")
			grpckit.PutBuffer(buf)
		}
	})

	// Concurrent access - tests pool contention
	b.Run("Concurrent_4Goroutines", func(b *testing.B) {
		b.ReportAllocs()
		var wg sync.WaitGroup
		work := b.N / 4
		if work == 0 {
			work = 1
		}

		for g := 0; g < 4; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < work; i++ {
					buf := grpckit.GetBuffer()
					buf.WriteString("data")
					grpckit.PutBuffer(buf)
				}
			}()
		}
		wg.Wait()
	})

	// Burst pattern - many gets before puts
	b.Run("Burst", func(b *testing.B) {
		b.ReportAllocs()
		bufs := make([]*bytes.Buffer, 10)
		for i := 0; i < b.N; i++ {
			// Get burst
			for j := 0; j < 10; j++ {
				bufs[j] = grpckit.GetBuffer()
				bufs[j].WriteString("data")
			}
			// Put burst
			for j := 0; j < 10; j++ {
				grpckit.PutBuffer(bufs[j])
			}
		}
	})
}

// BenchmarkBufferPool_VaryingSize benchmarks with different buffer sizes.
func BenchmarkBufferPool_VaryingSize(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"64B", 64},
		{"1KB", 1024},
		{"4KB", 4096},
		{"16KB", 16384},
		{"64KB", 65536},  // At the limit
		{"128KB", 131072}, // Over the limit - should not be pooled
	}

	for _, tc := range sizes {
		data := bytes.Repeat([]byte("x"), tc.size)

		b.Run(tc.name+"_Pooled", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				buf := grpckit.GetBuffer()
				buf.Write(data)
				grpckit.PutBuffer(buf)
			}
		})
	}
}
