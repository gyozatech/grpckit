# gRPCkit Benchmark Makefile
# Run benchmarks and generate reports

.PHONY: all bench bench-micro bench-integration bench-compare bench-profile bench-report proto clean help

# Default benchmark flags
BENCH_FLAGS ?= -benchmem -count=3
BENCH_TIME ?= 1s
BENCH_CPU ?= 1,2,4

# Output directories
RESULTS_DIR := results
PROFILE_DIR := profiles

all: bench

# Create output directories
$(RESULTS_DIR) $(PROFILE_DIR):
	mkdir -p $@

# Generate proto files
proto:
	cd proto && buf generate

# Run all benchmarks
bench: bench-micro bench-integration bench-compare

# Run microbenchmarks (internal function performance)
bench-micro: $(RESULTS_DIR)
	@echo "=== Running Microbenchmarks ==="
	go test $(BENCH_FLAGS) -benchtime=$(BENCH_TIME) -bench=. ./micro/... | tee $(RESULTS_DIR)/micro.txt

# Run integration benchmarks (end-to-end gRPCkit performance)
bench-integration: $(RESULTS_DIR)
	@echo "=== Running Integration Benchmarks ==="
	go test $(BENCH_FLAGS) -benchtime=$(BENCH_TIME) -bench=. ./integration/... | tee $(RESULTS_DIR)/integration.txt

# Run comparison benchmarks (cross-library)
bench-compare: $(RESULTS_DIR)
	@echo "=== Running Comparison Benchmarks ==="
	go test $(BENCH_FLAGS) -benchtime=$(BENCH_TIME) -bench=. ./comparison/... | tee $(RESULTS_DIR)/comparison.txt

# Run quick benchmarks (shorter duration)
bench-quick:
	@echo "=== Running Quick Benchmarks ==="
	go test -benchmem -benchtime=100ms -bench=. ./...

# Run parallel benchmarks with different CPU counts
bench-parallel: $(RESULTS_DIR)
	@echo "=== Running Parallel Benchmarks ==="
	@for cpu in $(BENCH_CPU); do \
		echo "--- CPU count: $$cpu ---"; \
		GOMAXPROCS=$$cpu go test $(BENCH_FLAGS) -benchtime=$(BENCH_TIME) -bench=Parallel ./... | tee $(RESULTS_DIR)/parallel_cpu$$cpu.txt; \
	done

# CPU profiling
bench-profile-cpu: $(PROFILE_DIR)
	@echo "=== Running CPU Profile ==="
	go test -bench=BenchmarkUnaryCall_gRPC -cpuprofile=$(PROFILE_DIR)/cpu.prof ./integration/...
	@echo "View with: go tool pprof $(PROFILE_DIR)/cpu.prof"

# Memory profiling
bench-profile-mem: $(PROFILE_DIR)
	@echo "=== Running Memory Profile ==="
	go test -bench=BenchmarkUnaryCall_gRPC -memprofile=$(PROFILE_DIR)/mem.prof ./integration/...
	@echo "View with: go tool pprof $(PROFILE_DIR)/mem.prof"

# Block profiling
bench-profile-block: $(PROFILE_DIR)
	@echo "=== Running Block Profile ==="
	go test -bench=BenchmarkUnaryCall_Parallel -blockprofile=$(PROFILE_DIR)/block.prof ./integration/...
	@echo "View with: go tool pprof $(PROFILE_DIR)/block.prof"

# All profiling
bench-profile: bench-profile-cpu bench-profile-mem

# Generate benchstat comparison report
bench-report: $(RESULTS_DIR)
	@echo "=== Generating Benchmark Report ==="
	@if command -v benchstat >/dev/null 2>&1; then \
		echo "--- Micro Benchmarks ---"; \
		benchstat $(RESULTS_DIR)/micro.txt 2>/dev/null || true; \
		echo ""; \
		echo "--- Integration Benchmarks ---"; \
		benchstat $(RESULTS_DIR)/integration.txt 2>/dev/null || true; \
		echo ""; \
		echo "--- Comparison Benchmarks ---"; \
		benchstat $(RESULTS_DIR)/comparison.txt 2>/dev/null || true; \
	else \
		echo "benchstat not found. Install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
		cat $(RESULTS_DIR)/*.txt; \
	fi

# Compare two benchmark runs (usage: make bench-diff OLD=results/old.txt NEW=results/new.txt)
bench-diff:
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Usage: make bench-diff OLD=results/old.txt NEW=results/new.txt"; \
		exit 1; \
	fi
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat $(OLD) $(NEW); \
	else \
		echo "benchstat not found. Install with: go install golang.org/x/perf/cmd/benchstat@latest"; \
	fi

# Run specific benchmark pattern (usage: make bench-run PATTERN=BenchmarkNormalizePath)
bench-run:
	@if [ -z "$(PATTERN)" ]; then \
		echo "Usage: make bench-run PATTERN=BenchmarkNormalizePath"; \
		exit 1; \
	fi
	go test $(BENCH_FLAGS) -benchtime=$(BENCH_TIME) -bench=$(PATTERN) ./...

# Install benchmark dependencies
deps:
	go mod tidy
	@echo "Installing benchstat..."
	go install golang.org/x/perf/cmd/benchstat@latest

# Clean generated files and results
clean:
	rm -rf $(RESULTS_DIR) $(PROFILE_DIR)
	rm -f proto/gen/*.go proto/gen/**/*.go

# Show help
help:
	@echo "gRPCkit Benchmark Suite"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              Run all benchmarks (default)"
	@echo "  bench            Run all benchmarks"
	@echo "  bench-micro      Run microbenchmarks (internal functions)"
	@echo "  bench-integration Run integration benchmarks (end-to-end)"
	@echo "  bench-compare    Run comparison benchmarks (vs other frameworks)"
	@echo "  bench-quick      Run quick benchmarks (100ms per test)"
	@echo "  bench-parallel   Run parallel benchmarks with different CPU counts"
	@echo "  bench-profile    Run CPU and memory profiling"
	@echo "  bench-profile-cpu Run CPU profiling"
	@echo "  bench-profile-mem Run memory profiling"
	@echo "  bench-report     Generate benchmark report with benchstat"
	@echo "  bench-diff       Compare two benchmark runs"
	@echo "  bench-run        Run specific benchmark pattern"
	@echo "  proto            Regenerate proto files"
	@echo "  deps             Install dependencies"
	@echo "  clean            Clean generated files and results"
	@echo ""
	@echo "Variables:"
	@echo "  BENCH_FLAGS      Benchmark flags (default: -benchmem -count=3)"
	@echo "  BENCH_TIME       Time per benchmark (default: 1s)"
	@echo "  BENCH_CPU        CPU counts for parallel tests (default: 1,2,4)"
	@echo ""
	@echo "Examples:"
	@echo "  make bench-micro                         # Run microbenchmarks"
	@echo "  make bench-run PATTERN=BenchmarkNormalizePath"
	@echo "  make bench-diff OLD=old.txt NEW=new.txt  # Compare runs"
	@echo "  make BENCH_TIME=5s bench-compare         # Longer benchmark runs"
