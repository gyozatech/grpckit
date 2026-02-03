package micro

import (
	"testing"

	"github.com/gyozatech/grpckit"
)

// BenchmarkNormalizePath benchmarks path normalization for metrics labels.
func BenchmarkNormalizePath(b *testing.B) {
	cases := []struct {
		name string
		path string
	}{
		{"Static", "/api/v1/users"},
		{"NumericID", "/api/v1/users/12345"},
		{"UUID", "/api/v1/users/550e8400-e29b-41d4-a716-446655440000"},
		{"Base64Like", "/api/v1/tokens/aGVsbG8td29ybGQtYmFzZTY0LWxpa2U"},
		{"NestedNumeric", "/api/v1/users/123/posts/456"},
		{"NestedUUID", "/api/v1/users/550e8400-e29b-41d4-a716-446655440000/items/6ba7b810-9dad-11d1-80b4-00c04fd430c8"},
		{"Root", "/"},
		{"Empty", ""},
		{"Health", "/healthz"},
		{"Metrics", "/metrics"},
		{"DeepPath", "/api/v1/organizations/123/teams/456/members/789/roles"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = grpckit.NormalizePath(tc.path)
			}
		})
	}
}

// BenchmarkNormalizePath_Parallel benchmarks path normalization under concurrent load.
func BenchmarkNormalizePath_Parallel(b *testing.B) {
	paths := []string{
		"/api/v1/users/12345",
		"/api/v1/items/550e8400-e29b-41d4-a716-446655440000",
		"/api/v1/orders/aGVsbG8td29ybGQtYmFzZTY0",
		"/api/v1/products/999",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = grpckit.NormalizePath(paths[i%len(paths)])
			i++
		}
	})
}

// BenchmarkIsLikelyID benchmarks ID detection for various patterns.
func BenchmarkIsLikelyID(b *testing.B) {
	cases := []struct {
		name  string
		input string
	}{
		{"Empty", ""},
		{"Short", "abc"},
		{"NumericSmall", "123"},
		{"NumericLarge", "9223372036854775807"},
		{"UUID", "550e8400-e29b-41d4-a716-446655440000"},
		{"Base64Short", "aGVsbG8"},                         // < 20 chars
		{"Base64Long", "aGVsbG8td29ybGQtYmFzZTY0LWxpa2U="}, // >= 20 chars
		{"Word", "users"},
		{"Path", "api/v1"},
		{"Mixed", "user_123_abc"},
		{"Hyphenated", "my-resource-name"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = grpckit.IsLikelyID(tc.input)
			}
		})
	}
}

// BenchmarkIsLikelyID_Parallel benchmarks ID detection under concurrent load.
func BenchmarkIsLikelyID_Parallel(b *testing.B) {
	inputs := []string{
		"12345",
		"550e8400-e29b-41d4-a716-446655440000",
		"aGVsbG8td29ybGQtYmFzZTY0LWxpa2U=",
		"users",
		"api",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = grpckit.IsLikelyID(inputs[i%len(inputs)])
			i++
		}
	})
}
