package statemachine

// Command represents a state machine command
type Command struct {
	Type CommandType
	Key  string
	Value []byte
}

// CommandType defines the type of command
type CommandType int

const (
	// CommandPut sets a key-value pair
	CommandPut CommandType = iota
	// CommandDelete removes a key
	CommandDelete
	// CommandGet retrieves a value (for read-only queries)
	CommandGet
)

// CommandResponse represents the result of applying a command
type CommandResponse struct {
	Success bool
	Value   []byte
	Error   error
}

// StateMachine defines the interface for a replicated state machine
// All operations must be deterministic
type StateMachine interface {
	// Apply applies a committed log entry to the state machine
	// This must be deterministic - same input always produces same output
	Apply(entry []byte) (interface{}, error)

	// Snapshot creates a point-in-time snapshot of the state machine
	// Returns the snapshot data
	Snapshot() ([]byte, error)

	// Restore restores the state machine from a snapshot
	Restore(snapshot []byte) error

	// Get retrieves a value from the state machine (read-only)
	// This doesn't go through the Raft log
	Get(key string) ([]byte, bool)
}
