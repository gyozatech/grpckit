package micro

import (
	"testing"

	"github.com/gyozatech/grpckit"
)

// BenchmarkExtractToken benchmarks token extraction from Authorization header.
func BenchmarkExtractToken(b *testing.B) {
	cases := []struct {
		name   string
		header string
	}{
		{"Empty", ""},
		{"ShortToken", "abc"},
		{"BearerLower", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"BearerUpper", "BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"BearerMixed", "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"BearerUncommon", "BeArEr eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"NoBearer", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"Basic", "Basic dXNlcjpwYXNz"},
		{"LongJWT", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = grpckit.ExtractToken(tc.header)
			}
		})
	}
}

// BenchmarkExtractToken_Parallel benchmarks token extraction under concurrent load.
func BenchmarkExtractToken_Parallel(b *testing.B) {
	headers := []string{
		"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"BEARER eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = grpckit.ExtractToken(headers[i%len(headers)])
			i++
		}
	})
}

// BenchmarkExtractToken_RealisticWorkload simulates realistic auth header processing.
func BenchmarkExtractToken_RealisticWorkload(b *testing.B) {
	// Realistic distribution: 90% standard Bearer, 5% lowercase, 3% no token, 2% other
	headers := make([]string, 100)
	for i := 0; i < 90; i++ {
		headers[i] = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ"
	}
	for i := 90; i < 95; i++ {
		headers[i] = "bearer token-" + string(rune('a'+i))
	}
	for i := 95; i < 98; i++ {
		headers[i] = ""
	}
	for i := 98; i < 100; i++ {
		headers[i] = "Basic dXNlcjpwYXNz"
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = grpckit.ExtractToken(headers[i%len(headers)])
	}
}
