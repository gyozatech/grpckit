package grpckit

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrUnauthorized",
			err:      ErrUnauthorized,
			expected: "unauthorized: missing or invalid token",
		},
		{
			name:     "ErrForbidden",
			err:      ErrForbidden,
			expected: "forbidden: insufficient permissions",
		},
		{
			name:     "ErrInvalidConfig",
			err:      ErrInvalidConfig,
			expected: "invalid configuration",
		},
		{
			name:     "ErrServiceNotRegistered",
			err:      ErrServiceNotRegistered,
			expected: "no services registered",
		},
		{
			name:     "ErrNotFound",
			err:      ErrNotFound,
			expected: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.err.Error())
			}
		})
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	errs := []error{
		ErrUnauthorized,
		ErrForbidden,
		ErrInvalidConfig,
		ErrServiceNotRegistered,
		ErrNotFound,
	}

	for i, err1 := range errs {
		for j, err2 := range errs {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("errors should be distinct: %v == %v", err1, err2)
			}
		}
	}
}

func TestErrorsCanBeWrapped(t *testing.T) {
	wrapped := errors.New("context: " + ErrUnauthorized.Error())
	if wrapped.Error() != "context: unauthorized: missing or invalid token" {
		t.Errorf("unexpected wrapped error: %v", wrapped)
	}
}
