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
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var notFoundErr *NotFoundError
	return errors.As(err, &notFoundErr)
}
