package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("charts/my-app", "main")

	expected := "charts/my-app not found at ref main"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "typed NotFoundError",
			err:  NewNotFoundError("resource", "ref"),
			want: true,
		},
		{
			name: "wrapped NotFoundError",
			err:  fmt.Errorf("failed to fetch: %w", NewNotFoundError("resource", "ref")),
			want: true,
		},
		{
			name: "generic error with 'not found' message",
			err:  errors.New("file not found in archive"),
			want: true,
		},
		{
			name: "generic error with 'no such file' message",
			err:  errors.New("no such file or directory"),
			want: true,
		},
		{
			name: "generic error with '404' message",
			err:  errors.New("HTTP 404 Not Found"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("permission denied"),
			want: false,
		},
		{
			name: "empty error message",
			err:  errors.New(""),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
