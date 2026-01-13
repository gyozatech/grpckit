package grpckit

import "errors"

// Common errors returned by grpckit.
var (
	// ErrUnauthorized is returned when authentication fails or token is missing.
	ErrUnauthorized = errors.New("unauthorized: missing or invalid token")

	// ErrForbidden is returned when the user doesn't have permission.
	ErrForbidden = errors.New("forbidden: insufficient permissions")

	// ErrInvalidConfig is returned when configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrServiceNotRegistered is returned when no services are registered.
	ErrServiceNotRegistered = errors.New("no services registered")

	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not found")
)
