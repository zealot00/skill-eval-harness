package apiclient

import (
	"errors"
	"fmt"
)

var (
	// ErrRemoteDisabled indicates no remote API config was provided.
	ErrRemoteDisabled = errors.New("remote api is disabled")
	// ErrNotFound indicates resource does not exist.
	ErrNotFound = errors.New("resource not found")
)

// APIError wraps a structured API error payload.
type APIError struct {
	StatusCode int
	Payload    APIErrorPayload
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Payload.Message == "" {
		return fmt.Sprintf("api error (status=%d)", e.StatusCode)
	}
	return fmt.Sprintf("api error (status=%d): %s", e.StatusCode, e.Payload.Message)
}

func (e *APIError) Is(target error) bool {
	if target == ErrNotFound {
		return e != nil && e.StatusCode == 404
	}
	return false
}
