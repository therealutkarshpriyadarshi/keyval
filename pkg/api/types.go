package api

import "time"

// Request represents a client request to the key-value store
type Request struct {
	// Unique request ID for idempotency
	ID string

	// Client ID
	ClientID string

	// Sequence number for this client
	Sequence uint64

	// Operation type
	Type OperationType

	// Key for the operation
	Key string

	// Value for Put operations
	Value []byte

	// Timeout for the request
	Timeout time.Duration
}

// OperationType defines the type of operation
type OperationType int

const (
	// OperationPut sets a key-value pair
	OperationPut OperationType = iota
	// OperationGet retrieves a value
	OperationGet
	// OperationDelete removes a key
	OperationDelete
)

// String returns the string representation of an operation type
func (o OperationType) String() string {
	switch o {
	case OperationPut:
		return "PUT"
	case OperationGet:
		return "GET"
	case OperationDelete:
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}

// Response represents the response to a client request
type Response struct {
	// Success indicates if the operation succeeded
	Success bool

	// Value returned for Get operations
	Value []byte

	// Error message if operation failed
	Error string

	// LeaderID if this node is not the leader (for redirection)
	LeaderID string

	// LeaderAddress for client redirection
	LeaderAddress string
}

// Error codes
const (
	ErrCodeNotLeader      = "not_leader"
	ErrCodeKeyNotFound    = "key_not_found"
	ErrCodeTimeout        = "timeout"
	ErrCodeInternalError  = "internal_error"
	ErrCodeInvalidRequest = "invalid_request"
)
