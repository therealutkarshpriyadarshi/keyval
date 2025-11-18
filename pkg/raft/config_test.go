package raft

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if err := config.Validate(); err != nil {
		t.Errorf("DefaultConfig validation failed: %v", err)
	}

	// Check some values
	if config.ElectionTimeoutMin != 150*time.Millisecond {
		t.Errorf("ElectionTimeoutMin = %v, want 150ms", config.ElectionTimeoutMin)
	}

	if config.ElectionTimeoutMax != 300*time.Millisecond {
		t.Errorf("ElectionTimeoutMax = %v, want 300ms", config.ElectionTimeoutMax)
	}

	if config.HeartbeatInterval != 50*time.Millisecond {
		t.Errorf("HeartbeatInterval = %v, want 50ms", config.HeartbeatInterval)
	}
}

func TestFastConfig(t *testing.T) {
	config := FastConfig()

	if err := config.Validate(); err != nil {
		t.Errorf("FastConfig validation failed: %v", err)
	}

	// Fast config should have shorter timeouts
	if config.ElectionTimeoutMin >= 150*time.Millisecond {
		t.Errorf("FastConfig ElectionTimeoutMin should be less than 150ms, got %v", config.ElectionTimeoutMin)
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()

	if err := config.Validate(); err != nil {
		t.Errorf("ProductionConfig validation failed: %v", err)
	}

	// Production config should have longer timeouts
	if config.ElectionTimeoutMin <= 150*time.Millisecond {
		t.Errorf("ProductionConfig ElectionTimeoutMin should be greater than 150ms, got %v", config.ElectionTimeoutMin)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid: zero election timeout min",
			config: &Config{
				ElectionTimeoutMin: 0,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  50 * time.Millisecond,
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: max less than min",
			config: &Config{
				ElectionTimeoutMin: 300 * time.Millisecond,
				ElectionTimeoutMax: 150 * time.Millisecond,
				HeartbeatInterval:  50 * time.Millisecond,
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: heartbeat too long",
			config: &Config{
				ElectionTimeoutMin: 150 * time.Millisecond,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  100 * time.Millisecond, // >= 150/2
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: zero heartbeat interval",
			config: &Config{
				ElectionTimeoutMin: 150 * time.Millisecond,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  0,
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: zero RPC timeout",
			config: &Config{
				ElectionTimeoutMin: 150 * time.Millisecond,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  50 * time.Millisecond,
				RPCTimeout:         0,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: zero max entries",
			config: &Config{
				ElectionTimeoutMin: 150 * time.Millisecond,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  50 * time.Millisecond,
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 0,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  10000,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
		{
			name: "invalid: zero snapshot threshold",
			config: &Config{
				ElectionTimeoutMin: 150 * time.Millisecond,
				ElectionTimeoutMax: 300 * time.Millisecond,
				HeartbeatInterval:  50 * time.Millisecond,
				RPCTimeout:         1 * time.Second,
				MaxEntriesPerAppend: 100,
				SnapshotInterval:   30 * time.Second,
				SnapshotThreshold:  0,
				MaxAppendEntriesRetries: 3,
				ApplyChanBufferSize: 100,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_HeartbeatVsElectionTimeout(t *testing.T) {
	// Heartbeat should be significantly less than election timeout
	// to allow multiple heartbeats before timeout
	config := DefaultConfig()

	// Should have at least 2-3 heartbeats before minimum election timeout
	heartbeatsBeforeTimeout := config.ElectionTimeoutMin / config.HeartbeatInterval
	if heartbeatsBeforeTimeout < 2 {
		t.Errorf("Too few heartbeats (%v) before election timeout", heartbeatsBeforeTimeout)
	}
}
