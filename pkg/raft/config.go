package raft

import (
	"fmt"
	"time"
)

// Config represents the configuration for a Raft node
type Config struct {
	// ElectionTimeoutMin is the minimum election timeout
	// Default: 150ms (recommended by Raft paper)
	ElectionTimeoutMin time.Duration

	// ElectionTimeoutMax is the maximum election timeout
	// Default: 300ms (recommended by Raft paper)
	ElectionTimeoutMax time.Duration

	// HeartbeatInterval is the interval at which leaders send heartbeats
	// Must be significantly less than ElectionTimeoutMin
	// Default: 50ms (recommended by Raft paper)
	HeartbeatInterval time.Duration

	// RPCTimeout is the timeout for RPC calls
	// Default: 1 second
	RPCTimeout time.Duration

	// MaxEntriesPerAppend is the maximum number of log entries to send in a single AppendEntries RPC
	// Default: 100
	MaxEntriesPerAppend int

	// SnapshotInterval is the interval at which to check if a snapshot should be taken
	// Default: 30 seconds
	SnapshotInterval time.Duration

	// SnapshotThreshold is the number of log entries before triggering a snapshot
	// Default: 10000
	SnapshotThreshold uint64

	// MaxAppendEntriesRetries is the maximum number of times to retry AppendEntries
	// Default: 3
	MaxAppendEntriesRetries int

	// ApplyChanBufferSize is the size of the apply channel buffer
	// Default: 100
	ApplyChanBufferSize int
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		ElectionTimeoutMin:      150 * time.Millisecond,
		ElectionTimeoutMax:      300 * time.Millisecond,
		HeartbeatInterval:       50 * time.Millisecond,
		RPCTimeout:              1 * time.Second,
		MaxEntriesPerAppend:     100,
		SnapshotInterval:        30 * time.Second,
		SnapshotThreshold:       10000,
		MaxAppendEntriesRetries: 3,
		ApplyChanBufferSize:     100,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ElectionTimeoutMin <= 0 {
		return fmt.Errorf("ElectionTimeoutMin must be positive")
	}

	if c.ElectionTimeoutMax <= c.ElectionTimeoutMin {
		return fmt.Errorf("ElectionTimeoutMax must be greater than ElectionTimeoutMin")
	}

	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("HeartbeatInterval must be positive")
	}

	// Heartbeat interval should be significantly less than election timeout
	// to ensure heartbeats are sent multiple times before timeout
	if c.HeartbeatInterval >= c.ElectionTimeoutMin/2 {
		return fmt.Errorf("HeartbeatInterval should be less than ElectionTimeoutMin/2 (got %v, min is %v)",
			c.HeartbeatInterval, c.ElectionTimeoutMin/2)
	}

	if c.RPCTimeout <= 0 {
		return fmt.Errorf("RPCTimeout must be positive")
	}

	if c.MaxEntriesPerAppend <= 0 {
		return fmt.Errorf("MaxEntriesPerAppend must be positive")
	}

	if c.SnapshotInterval <= 0 {
		return fmt.Errorf("SnapshotInterval must be positive")
	}

	if c.SnapshotThreshold == 0 {
		return fmt.Errorf("SnapshotThreshold must be positive")
	}

	if c.MaxAppendEntriesRetries <= 0 {
		return fmt.Errorf("MaxAppendEntriesRetries must be positive")
	}

	if c.ApplyChanBufferSize <= 0 {
		return fmt.Errorf("ApplyChanBufferSize must be positive")
	}

	return nil
}

// FastConfig returns a Config optimized for testing with shorter timeouts
func FastConfig() *Config {
	return &Config{
		ElectionTimeoutMin:      50 * time.Millisecond,
		ElectionTimeoutMax:      100 * time.Millisecond,
		HeartbeatInterval:       10 * time.Millisecond,
		RPCTimeout:              500 * time.Millisecond,
		MaxEntriesPerAppend:     100,
		SnapshotInterval:        5 * time.Second,
		SnapshotThreshold:       1000,
		MaxAppendEntriesRetries: 3,
		ApplyChanBufferSize:     100,
	}
}

// ProductionConfig returns a Config optimized for production use with conservative timeouts
func ProductionConfig() *Config {
	return &Config{
		ElectionTimeoutMin:      300 * time.Millisecond,
		ElectionTimeoutMax:      500 * time.Millisecond,
		HeartbeatInterval:       100 * time.Millisecond,
		RPCTimeout:              2 * time.Second,
		MaxEntriesPerAppend:     500,
		SnapshotInterval:        60 * time.Second,
		SnapshotThreshold:       50000,
		MaxAppendEntriesRetries: 5,
		ApplyChanBufferSize:     200,
	}
}
