# gRPCkit Benchmark Suite

Comprehensive benchmark suite for measuring gRPCkit's performance and comparing it against similar Go frameworks for dual gRPC/REST microservices.

## Quick Start

```bash
# Install dependencies
make deps

# Run all benchmarks
make bench

# Run specific benchmark category
make bench-micro       # Internal function benchmarks
make bench-integration # End-to-end benchmarks
make bench-compare     # Cross-library comparisons
```

## Directory Structure

```
benchmark/
├── go.mod                  # Separate module for benchmark deps
├── Makefile                # Benchmark automation
├── README.md               # This file
│
├── proto/
│   ├── benchmark.proto     # Shared service definition
│   └── gen/                # Generated Go code
│
├── internal/
│   ├── service/
│   │   └── impl.go         # Shared service implementation
│   └── harness/
│       ├── common.go       # Common harness interface
│       ├── grpckit.go      # gRPCkit test harness
│       ├── vanilla.go      # Standard grpc + grpc-gateway
│       ├── connect.go      # Connect-go harness
│       ├── twirp.go        # Twirp harness
│       └── gin.go          # Gin + grpc-gateway harness
│
├── micro/                  # Microbenchmarks (isolated functions)
│   ├── path_normalize_test.go
│   ├── auth_matching_test.go
│   ├── token_extract_test.go
│   └── buffer_pool_test.go
│
├── integration/            # Integration benchmarks
│   ├── unary_test.go
│   ├── auth_test.go
│   ├── middleware_test.go
│   └── health_test.go
│
└── comparison/             # Cross-library comparison
    ├── grpc_test.go
    ├── rest_test.go
    └── throughput_test.go
```

## Benchmark Categories

### Microbenchmarks (`micro/`)

Target the optimized code paths from recent performance improvements:

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkNormalizePath` | Path normalization for metrics labels |
| `BenchmarkIsLikelyID` | ID detection (numeric, UUID, base64) |
| `BenchmarkMatchPattern` | Glob pattern matching |
| `BenchmarkMatchesCompiledPatterns` | O(1) exact match + wildcard patterns |
| `BenchmarkExtractToken` | Bearer token extraction (allocation-free) |
| `BenchmarkBufferPool` | Buffer pool get/put operations |

### Integration Benchmarks (`integration/`)

End-to-end benchmarks for gRPCkit features:

| Benchmark | Protocol | Features |
|-----------|----------|----------|
| `BenchmarkUnaryCall_gRPC` | gRPC | Baseline gRPC latency |
| `BenchmarkUnaryCall_REST` | REST | REST via grpc-gateway |
| `BenchmarkAuth_*` | Both | Authentication overhead |
| `BenchmarkMiddleware_*` | REST | Metrics/CORS middleware |
| `BenchmarkHealth_*` | REST | Health endpoint throughput |
| `BenchmarkPayloadSizes` | Both | Different payload sizes |

### Comparison Benchmarks (`comparison/`)

Cross-library performance comparison:

| Framework | Description |
|-----------|-------------|
| **gRPCkit** | This framework |
| **Vanilla** | Manual grpc + grpc-gateway |
| **Connect-go** | Buf's modern gRPC alternative |
| **Twirp** | Twitch's simple RPC framework |
| **Gin** | Popular HTTP router + grpc-gateway |

## Running Benchmarks

### Basic Commands

```bash
# Run all benchmarks
make bench

# Run with custom settings
make BENCH_TIME=5s BENCH_FLAGS="-benchmem -count=5" bench

# Run specific pattern
make bench-run PATTERN=BenchmarkNormalizePath

# Quick smoke test
make bench-quick
```

### Profiling

```bash
# CPU profiling
make bench-profile-cpu
go tool pprof profiles/cpu.prof

# Memory profiling
make bench-profile-mem
go tool pprof profiles/mem.prof

# Block profiling (goroutine contention)
make bench-profile-block
```

### Comparing Runs

```bash
# Save baseline
make bench > results/baseline.txt

# Make changes, then compare
make bench > results/new.txt
make bench-diff OLD=results/baseline.txt NEW=results/new.txt
```

## Metrics Collected

| Metric | Description |
|--------|-------------|
| **ns/op** | Nanoseconds per operation (latency) |
| **B/op** | Bytes allocated per operation |
| **allocs/op** | Number of allocations per operation |
| **ops/sec** | Operations per second (throughput) |

## Expected Results

Based on gRPCkit's optimizations:

### Path Normalization
- Static paths: ~50ns/op, 0 allocs
- Paths with IDs: ~100-200ns/op, 1-2 allocs

### Auth Pattern Matching
- Exact match (O(1) map lookup): ~5-10ns/op
- Wildcard patterns: ~20-50ns/op

### Token Extraction
- Standard "Bearer " prefix: ~2-5ns/op, 0 allocs
- Uncommon casing: ~50ns/op (fallback to ToLower)

### Buffer Pooling
- Pool hit: ~10-20ns/op vs ~100ns/op without pool
- 50-70% reduction in GC pressure under load

### Framework Comparison (relative)
| Framework | gRPC Latency | REST Latency |
|-----------|--------------|--------------|
| gRPCkit | 1.0x (baseline) | 1.0x |
| Vanilla | ~1.0x | ~0.95x |
| Connect | ~0.9x | N/A |
| Gin | ~1.0x | ~1.05x |

## Troubleshooting

### "benchstat not found"
```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

### Proto generation fails
```bash
# Ensure buf is installed
brew install bufbuild/buf/buf  # or equivalent

# Update dependencies
cd proto && buf dep update
```

### High variance in results
- Use longer benchmark times: `make BENCH_TIME=5s bench`
- Increase count: `make BENCH_FLAGS="-benchmem -count=10" bench`
- Close other applications
- Disable CPU frequency scaling if possible

## Adding New Benchmarks

1. Create test file in appropriate directory (`micro/`, `integration/`, or `comparison/`)
2. Follow naming convention: `Benchmark<Category>_<Variant>`
3. Always use `b.ReportAllocs()` for memory tracking
4. Use `b.ResetTimer()` after setup code
5. For comparisons, create harness once outside the benchmark loop

Example:
```go
func BenchmarkMyFeature(b *testing.B) {
    // Setup
    h, _ := harness.NewGRPCkitHarness(...)
    defer h.Close()

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        // Benchmark code
    }
}
```

## License

Same as the parent gRPCkit project.
