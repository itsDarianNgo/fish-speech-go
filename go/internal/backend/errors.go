package backend

import (
	"errors"
	"fmt"
)

// ErrBackendUnavailable indicates the Python backend is not reachable.
var ErrBackendUnavailable = errors.New("backend unavailable")

// ErrBackendTimeout indicates the backend took too long to respond.
var ErrBackendTimeout = errors.New("backend timeout")

// BackendError represents an error returned by the Python backend.
type BackendError struct {
	StatusCode int
	Message    string
}

func (e *BackendError) Error() string {
	return fmt.Sprintf("backend error (status %d): %s", e.StatusCode, e.Message)
}

// IsBackendError checks if an error is a BackendError.
func IsBackendError(err error) bool {
	var be *BackendError
	return errors.As(err, &be)
}
