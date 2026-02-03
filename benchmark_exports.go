package grpckit

import "bytes"

// This file exports internal functions for benchmarking.
// These exports are only used by the benchmark package.

// NormalizePath normalizes URL paths for metrics labels.
// Exported for benchmarking purposes.
func NormalizePath(path string) string {
	return normalizePath(path)
}

// IsLikelyID checks if a path segment looks like a dynamic ID.
// Exported for benchmarking purposes.
func IsLikelyID(s string) bool {
	return isLikelyID(s)
}

// ExtractToken extracts the token from the Authorization header.
// Exported for benchmarking purposes.
func ExtractToken(header string) string {
	return extractToken(header)
}

// MatchPattern matches a path against a glob pattern.
// Exported for benchmarking purposes.
func MatchPattern(pattern, urlPath string) bool {
	return matchPattern(pattern, urlPath)
}

// CompiledPattern represents a pre-compiled pattern for efficient matching.
// Exported for benchmarking purposes.
type CompiledPattern = compiledPattern

// NewCompiledPattern creates a new compiled pattern for benchmarking.
func NewCompiledPattern(prefix, pattern string, isDouble bool) CompiledPattern {
	return compiledPattern{
		prefix:   prefix,
		pattern:  pattern,
		isDouble: isDouble,
	}
}

// MatchesCompiledPatterns checks if a path matches any compiled patterns.
// Exported for benchmarking purposes.
func MatchesCompiledPatterns(urlPath string, exactMap map[string]bool, wildcards []CompiledPattern) bool {
	return matchesCompiledPatterns(urlPath, exactMap, wildcards)
}

// GetBuffer retrieves a buffer from the pool.
// Exported for benchmarking purposes.
func GetBuffer() *bytes.Buffer {
	return getBuffer()
}

// PutBuffer returns a buffer to the pool.
// Exported for benchmarking purposes.
func PutBuffer(buf *bytes.Buffer) {
	putBuffer(buf)
}
