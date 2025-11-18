package raft

import (
	"testing"
	"time"
)

func TestNewHeartbeatTimer(t *testing.T) {
	interval := 50 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	if timer == nil {
		t.Fatal("NewHeartbeatTimer returned nil")
	}

	if timer.interval != interval {
		t.Errorf("Expected interval %v, got %v", interval, timer.interval)
	}

	if timer.running {
		t.Error("New timer should not be running")
	}
}

func TestHeartbeatTimerStartStop(t *testing.T) {
	interval := 10 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	// Start timer
	timer.Start()

	// Wait a bit to ensure it's running
	time.Sleep(5 * time.Millisecond)

	timer.mu.Lock()
	running := timer.running
	timer.mu.Unlock()

	if !running {
		t.Error("Timer should be running after Start()")
	}

	// Stop timer
	timer.Stop()

	timer.mu.Lock()
	running = timer.running
	timer.mu.Unlock()

	if running {
		t.Error("Timer should not be running after Stop()")
	}
}

func TestHeartbeatTimerTicks(t *testing.T) {
	interval := 20 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	timer.Start()
	defer timer.Stop()

	// Wait for at least 2 ticks
	tickCount := 0
	timeout := time.After(100 * time.Millisecond)

	for tickCount < 2 {
		select {
		case <-timer.C():
			tickCount++
		case <-timeout:
			t.Fatalf("Timeout waiting for ticks, got %d ticks", tickCount)
		}
	}

	if tickCount < 2 {
		t.Errorf("Expected at least 2 ticks, got %d", tickCount)
	}
}

func TestHeartbeatTimerReset(t *testing.T) {
	interval := 50 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	timer.Start()
	defer timer.Stop()

	// Wait for first tick
	select {
	case <-timer.C():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for first tick")
	}

	// Reset timer
	timer.Reset()

	// Should get another tick after the interval
	select {
	case <-timer.C():
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for tick after reset")
	}
}

func TestHeartbeatTimerMultipleStartStop(t *testing.T) {
	interval := 10 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	// Multiple starts should be safe
	timer.Start()
	timer.Start()
	timer.Start()

	timer.mu.Lock()
	running := timer.running
	timer.mu.Unlock()

	if !running {
		t.Error("Timer should be running")
	}

	// Multiple stops should be safe
	timer.Stop()
	timer.Stop()
	timer.Stop()

	timer.mu.Lock()
	running = timer.running
	timer.mu.Unlock()

	if running {
		t.Error("Timer should not be running")
	}
}

func TestHeartbeatTimerChannel(t *testing.T) {
	interval := 20 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	c := timer.C()
	if c == nil {
		t.Fatal("Timer channel should not be nil")
	}

	timer.Start()
	defer timer.Stop()

	// Should receive from channel
	select {
	case <-c:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for channel receive")
	}
}

func TestGetTermAtIndex(t *testing.T) {
	state := NewServerState()

	// Empty log
	term := state.GetTermAtIndex(0)
	if term != 0 {
		t.Errorf("GetTermAtIndex(0) on empty log should return 0, got %d", term)
	}

	term = state.GetTermAtIndex(1)
	if term != 0 {
		t.Errorf("GetTermAtIndex(1) on empty log should return 0, got %d", term)
	}

	// Add entries
	state.AppendEntry(LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal})
	state.AppendEntry(LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal})
	state.AppendEntry(LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal})
	state.AppendEntry(LogEntry{Term: 3, Index: 4, Data: []byte("cmd4"), Type: EntryNormal})

	tests := []struct {
		index        uint64
		expectedTerm uint64
	}{
		{0, 0},  // Index 0
		{1, 1},  // Valid
		{2, 1},  // Valid
		{3, 2},  // Valid
		{4, 3},  // Valid
		{5, 0},  // Out of bounds
		{10, 0}, // Out of bounds
	}

	for _, tt := range tests {
		term := state.GetTermAtIndex(tt.index)
		if term != tt.expectedTerm {
			t.Errorf("GetTermAtIndex(%d) = %d, expected %d", tt.index, term, tt.expectedTerm)
		}
	}
}

func TestHeartbeatTimerStopBeforeStart(t *testing.T) {
	interval := 10 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	// Stopping before starting should be safe
	timer.Stop()

	timer.mu.Lock()
	running := timer.running
	timer.mu.Unlock()

	if running {
		t.Error("Timer should not be running")
	}
}

func TestHeartbeatTimerConcurrentOperations(t *testing.T) {
	interval := 10 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	timer.Start()

	// Concurrent resets and stops
	done := make(chan bool)

	go func() {
		for i := 0; i < 10; i++ {
			timer.Reset()
			time.Sleep(2 * time.Millisecond)
		}
		done <- true
	}()

	go func() {
		time.Sleep(50 * time.Millisecond)
		timer.Stop()
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	timer.mu.Lock()
	running := timer.running
	timer.mu.Unlock()

	if running {
		t.Error("Timer should be stopped")
	}
}

// Test that heartbeat interval is appropriate relative to election timeout
func TestHeartbeatIntervalInConfig(t *testing.T) {
	config := DefaultConfig()

	// Heartbeat interval should be less than half of election timeout min
	if config.HeartbeatInterval >= config.ElectionTimeoutMin/2 {
		t.Errorf("HeartbeatInterval (%v) should be less than ElectionTimeoutMin/2 (%v)",
			config.HeartbeatInterval, config.ElectionTimeoutMin/2)
	}

	// Verify this constraint is enforced by validation
	invalidConfig := &Config{
		ElectionTimeoutMin:      100 * time.Millisecond,
		ElectionTimeoutMax:      200 * time.Millisecond,
		HeartbeatInterval:       60 * time.Millisecond, // Too large!
		RPCTimeout:              1 * time.Second,
		MaxEntriesPerAppend:     100,
		SnapshotInterval:        30 * time.Second,
		SnapshotThreshold:       10000,
		MaxAppendEntriesRetries: 3,
		ApplyChanBufferSize:     100,
	}

	err := invalidConfig.Validate()
	if err == nil {
		t.Error("Expected validation error for too-large heartbeat interval")
	}
}

func TestHeartbeatTimerTickRate(t *testing.T) {
	interval := 20 * time.Millisecond
	timer := NewHeartbeatTimer(interval)

	timer.Start()
	defer timer.Stop()

	// Measure time between ticks
	<-timer.C() // First tick

	start := time.Now()
	<-timer.C() // Second tick
	elapsed := time.Since(start)

	// Allow for some variance (±10ms)
	tolerance := 10 * time.Millisecond
	if elapsed < interval-tolerance || elapsed > interval+tolerance {
		t.Errorf("Tick interval was %v, expected around %v (±%v)", elapsed, interval, tolerance)
	}
}
