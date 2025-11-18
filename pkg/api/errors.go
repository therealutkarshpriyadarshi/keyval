package api

import "fmt"

// ErrorType represents different types of errors
type ErrorType int

const (
	// ErrorTypeNotLeader indicates the node is not the leader
	ErrorTypeNotLeader ErrorType = iota
	// ErrorTypeKeyNotFound indicates the key was not found
	ErrorTypeKeyNotFound
	// ErrorTypeTimeout indicates the operation timed out
	ErrorTypeTimeout
	// ErrorTypeInvalidRequest indicates the request was invalid
	ErrorTypeInvalidRequest
	// ErrorTypeInternalError indicates an internal error occurred
	ErrorTypeInternalError
	// ErrorTypeUnavailable indicates the cluster is unavailable
	ErrorTypeUnavailable
)

// Error represents a structured error from the API
type Error struct {
	Type    ErrorType
	Message string
	Details map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new API error
func NewError(errType ErrorType, message string) *Error {
	return &Error{
		Type:    errType,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

// WithDetail adds a detail to the error
func (e *Error) WithDetail(key string, value interface{}) *Error {
	e.Details[key] = value
	return e
}

// IsRetryable returns true if this error can be retried
func (e *Error) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeNotLeader, ErrorTypeTimeout, ErrorTypeUnavailable:
		return true
	default:
		return false
	}
}

// Common error constructors

// ErrNotLeaderError creates a not leader error
func ErrNotLeaderError(leaderID string) *Error {
	err := NewError(ErrorTypeNotLeader, "not the leader")
	if leaderID != "" {
		err.WithDetail("leader_id", leaderID)
	}
	return err
}

// ErrKeyNotFoundError creates a key not found error
func ErrKeyNotFoundError(key string) *Error {
	return NewError(ErrorTypeKeyNotFound, fmt.Sprintf("key not found: %s", key)).
		WithDetail("key", key)
}

// ErrTimeoutError creates a timeout error
func ErrTimeoutError(operation string) *Error {
	return NewError(ErrorTypeTimeout, fmt.Sprintf("operation timed out: %s", operation)).
		WithDetail("operation", operation)
}

// ErrInvalidRequestError creates an invalid request error
func ErrInvalidRequestError(reason string) *Error {
	return NewError(ErrorTypeInvalidRequest, fmt.Sprintf("invalid request: %s", reason)).
		WithDetail("reason", reason)
}

// ErrInternalError creates an internal error
func ErrInternalError(err error) *Error {
	return NewError(ErrorTypeInternalError, fmt.Sprintf("internal error: %v", err)).
		WithDetail("underlying_error", err.Error())
}

// ErrUnavailable creates an unavailable error
func ErrUnavailable() *Error {
	return NewError(ErrorTypeUnavailable, "cluster unavailable")
}
