package domain

import (
	"errors"
	"fmt"
)

// NotFoundError represents a resource that was not found at a specific ref.
type NotFoundError struct {
	Resource string
	Ref      string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found at ref %s", e.Resource, e.Ref)
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(resource, ref string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Ref:      ref,
	}
}

// IsNotFound checks if an error is or wraps a NotFoundError.
// It also checks for common "not found" error messages from external systems
// for backward compatibility.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's our typed error
	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return true
	}

	// Fallback: check error message for common "not found" patterns
	// This handles errors from GitHub API, filesystem, etc.
	msg := err.Error()
	return containsAny(msg, []string{"not found", "no such file or directory", "404"})
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
