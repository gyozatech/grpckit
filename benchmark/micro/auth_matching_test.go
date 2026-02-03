package micro

import (
	"testing"

	"github.com/gyozatech/grpckit"
)

// BenchmarkMatchPattern benchmarks glob pattern matching.
func BenchmarkMatchPattern(b *testing.B) {
	cases := []struct {
		name    string
		pattern string
		path    string
	}{
		{"ExactMatch", "/api/v1/users", "/api/v1/users"},
		{"ExactMismatch", "/api/v1/users", "/api/v1/items"},
		{"DoubleWildcard", "/api/**", "/api/v1/users/123"},
		{"SingleWildcard", "/api/*/users", "/api/v1/users"},
		{"NoMatch", "/admin/**", "/api/v1/users"},
		{"DeepPath", "/api/v1/organizations/*/teams/**", "/api/v1/organizations/123/teams/456/members"},
		{"RootWildcard", "/**", "/any/path/here"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = grpckit.MatchPattern(tc.pattern, tc.path)
			}
		})
	}
}

// BenchmarkMatchesCompiledPatterns benchmarks pre-compiled pattern matching.
func BenchmarkMatchesCompiledPatterns(b *testing.B) {
	// Simulate pre-compiled patterns as they would be created by WithProtectedEndpoints
	exactMap := map[string]bool{
		"/api/v1/users":    true,
		"/api/v1/items":    true,
		"/api/v1/orders":   true,
		"/api/v1/products": true,
		"/api/v1/settings": true,
	}

	wildcards := []grpckit.CompiledPattern{
		grpckit.NewCompiledPattern("/admin", "/admin/**", true),
		grpckit.NewCompiledPattern("/api/v2", "/api/v2/**", true),
	}

	cases := []struct {
		name string
		path string
	}{
		{"ExactMatch_First", "/api/v1/users"},
		{"ExactMatch_Last", "/api/v1/settings"},
		{"ExactMiss", "/api/v1/unknown"},
		{"WildcardMatch", "/admin/dashboard"},
		{"WildcardMiss", "/public/index.html"},
		{"DeepWildcard", "/api/v2/resources/123/items/456"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = grpckit.MatchesCompiledPatterns(tc.path, exactMap, wildcards)
			}
		})
	}
}

// BenchmarkMatchesCompiledPatterns_Parallel benchmarks pattern matching under concurrent load.
func BenchmarkMatchesCompiledPatterns_Parallel(b *testing.B) {
	exactMap := map[string]bool{
		"/api/v1/users":    true,
		"/api/v1/items":    true,
		"/api/v1/orders":   true,
		"/api/v1/products": true,
	}

	wildcards := []grpckit.CompiledPattern{
		grpckit.NewCompiledPattern("/admin", "/admin/**", true),
	}

	paths := []string{
		"/api/v1/users",
		"/api/v1/unknown",
		"/admin/dashboard",
		"/public/index.html",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = grpckit.MatchesCompiledPatterns(paths[i%len(paths)], exactMap, wildcards)
			i++
		}
	})
}

// BenchmarkMatchPattern_vs_CompiledPatterns compares raw pattern matching vs pre-compiled.
func BenchmarkMatchPattern_vs_CompiledPatterns(b *testing.B) {
	patterns := []string{
		"/api/v1/users",
		"/api/v1/items",
		"/api/v1/orders",
		"/admin/**",
	}

	exactMap := map[string]bool{
		"/api/v1/users":  true,
		"/api/v1/items":  true,
		"/api/v1/orders": true,
	}

	wildcards := []grpckit.CompiledPattern{
		grpckit.NewCompiledPattern("/admin", "/admin/**", true),
	}

	testPath := "/api/v1/users"

	b.Run("RawPatterns", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			for _, p := range patterns {
				if grpckit.MatchPattern(p, testPath) {
					break
				}
			}
		}
	})

	b.Run("CompiledPatterns", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = grpckit.MatchesCompiledPatterns(testPath, exactMap, wildcards)
		}
	})
}

// BenchmarkMatchesCompiledPatterns_LargeExactMap benchmarks with many exact patterns.
func BenchmarkMatchesCompiledPatterns_LargeExactMap(b *testing.B) {
	// Create a large exact map to test map lookup performance
	exactMap := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		exactMap["/api/v1/resource"+string(rune('A'+i%26))+string(rune('0'+i/26))] = true
	}
	exactMap["/api/v1/target"] = true // Add our target at the end

	wildcards := []grpckit.CompiledPattern{
		grpckit.NewCompiledPattern("/admin", "/admin/**", true),
	}

	b.Run("MapHit", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = grpckit.MatchesCompiledPatterns("/api/v1/target", exactMap, wildcards)
		}
	})

	b.Run("MapMiss_WildcardHit", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = grpckit.MatchesCompiledPatterns("/admin/dashboard", exactMap, wildcards)
		}
	})

	b.Run("AllMiss", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = grpckit.MatchesCompiledPatterns("/unknown/path", exactMap, wildcards)
		}
	})
}
