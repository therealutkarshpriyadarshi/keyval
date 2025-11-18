package statemachine

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"
)

// KVStateMachine is an in-memory key-value store that implements StateMachine
type KVStateMachine struct {
	mu    sync.RWMutex
	store map[string][]byte
}

// NewKVStateMachine creates a new key-value state machine
func NewKVStateMachine() *KVStateMachine {
	return &KVStateMachine{
		store: make(map[string][]byte),
	}
}

// Apply applies a command to the state machine
// The entry is expected to be a JSON-encoded Command
func (kv *KVStateMachine) Apply(entry []byte) (interface{}, error) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// Decode the command
	var cmd Command
	if err := json.Unmarshal(entry, &cmd); err != nil {
		return nil, fmt.Errorf("failed to decode command: %w", err)
	}

	// Apply the command
	switch cmd.Type {
	case CommandPut:
		kv.store[cmd.Key] = cmd.Value
		return &CommandResponse{
			Success: true,
			Value:   cmd.Value,
		}, nil

	case CommandDelete:
		delete(kv.store, cmd.Key)
		return &CommandResponse{
			Success: true,
		}, nil

	case CommandGet:
		// Get shouldn't go through Apply, but we handle it for completeness
		value, ok := kv.store[cmd.Key]
		if !ok {
			return &CommandResponse{
				Success: false,
				Error:   fmt.Errorf("key not found: %s", cmd.Key),
			}, nil
		}
		return &CommandResponse{
			Success: true,
			Value:   value,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command type: %d", cmd.Type)
	}
}

// Get retrieves a value from the state machine without going through Raft
// Returns the value and whether the key exists
func (kv *KVStateMachine) Get(key string) ([]byte, bool) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	value, ok := kv.store[key]
	if !ok {
		return nil, false
	}

	// Return a copy to avoid external modification
	result := make([]byte, len(value))
	copy(result, value)
	return result, true
}

// Snapshot creates a snapshot of the current state
func (kv *KVStateMachine) Snapshot() ([]byte, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	// Use gob encoding for the snapshot
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	// Encode the map
	if err := enc.Encode(kv.store); err != nil {
		return nil, fmt.Errorf("failed to encode snapshot: %w", err)
	}

	return buf.Bytes(), nil
}

// Restore restores the state machine from a snapshot
func (kv *KVStateMachine) Restore(snapshot []byte) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// Decode the snapshot
	buf := bytes.NewBuffer(snapshot)
	dec := gob.NewDecoder(buf)

	newStore := make(map[string][]byte)
	if err := dec.Decode(&newStore); err != nil {
		return fmt.Errorf("failed to decode snapshot: %w", err)
	}

	// Replace the store
	kv.store = newStore
	return nil
}

// Size returns the number of keys in the state machine
func (kv *KVStateMachine) Size() int {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	return len(kv.store)
}

// Keys returns all keys in the state machine
func (kv *KVStateMachine) Keys() []string {
	kv.mu.RLock()
	defer kv.mu.RUnlock()

	keys := make([]string, 0, len(kv.store))
	for k := range kv.store {
		keys = append(keys, k)
	}
	return keys
}

// Clear removes all entries from the state machine
func (kv *KVStateMachine) Clear() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store = make(map[string][]byte)
}
