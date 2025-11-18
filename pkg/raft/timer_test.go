package raft

import (
	"testing"
	"time"
)

func TestElectionTimer_Timeout(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	timer := NewElectionTimer(min, max)
	timer.Start()
	defer timer.Stop()

	start := time.Now()

	select {
	case <-timer.C():
		elapsed := time.Since(start)
		if elapsed < min || elapsed > max+50*time.Millisecond {
			t.Errorf("Timer fired after %v, expected between %v and %v", elapsed, min, max)
		}
	case <-time.After(max + 100*time.Millisecond):
		t.Error("Timer did not fire within expected time")
	}
}

func TestElectionTimer_Reset(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	timer := NewElectionTimer(min, max)
	timer.Start()
	defer timer.Stop()

	// Wait a bit
	time.Sleep(30 * time.Millisecond)

	// Reset the timer
	timer.Reset()

	start := time.Now()

	// Timer should fire after reset
	select {
	case <-timer.C():
		elapsed := time.Since(start)
		if elapsed < min || elapsed > max+50*time.Millisecond {
			t.Errorf("Timer fired after %v, expected between %v and %v", elapsed, min, max)
		}
	case <-time.After(max + 100*time.Millisecond):
		t.Error("Timer did not fire after reset")
	}
}

func TestElectionTimer_MultipleResets(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	timer := NewElectionTimer(min, max)
	timer.Start()
	defer timer.Stop()

	// Reset multiple times in quick succession
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		timer.Reset()
	}

	start := time.Now()

	// Timer should fire after the last reset
	select {
	case <-timer.C():
		elapsed := time.Since(start)
		if elapsed < min || elapsed > max+50*time.Millisecond {
			t.Errorf("Timer fired after %v, expected between %v and %v", elapsed, min, max)
		}
	case <-time.After(max + 100*time.Millisecond):
		t.Error("Timer did not fire after multiple resets")
	}
}

func TestElectionTimer_Stop(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	timer := NewElectionTimer(min, max)
	timer.Start()

	// Stop immediately
	timer.Stop()

	// Timer should not fire
	select {
	case <-timer.C():
		t.Error("Timer fired after being stopped")
	case <-time.After(max + 100*time.Millisecond):
		// Expected - timer should not fire
	}
}

func TestElectionTimer_Randomization(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	// Create multiple timers and check that they have different timeouts
	const numTimers = 10
	timeouts := make([]time.Duration, numTimers)

	for i := 0; i < numTimers; i++ {
		timer := NewElectionTimer(min, max)
		timer.Start()

		start := time.Now()
		<-timer.C()
		timeouts[i] = time.Since(start)
		timer.Stop()
	}

	// Check that not all timeouts are the same
	// (there's a very small chance they could all be the same by random chance)
	allSame := true
	first := timeouts[0]
	for i := 1; i < numTimers; i++ {
		if timeouts[i] != first {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("All timeouts were the same; randomization may not be working")
	}

	// Check that all timeouts are within bounds
	for i, timeout := range timeouts {
		if timeout < min || timeout > max+50*time.Millisecond {
			t.Errorf("Timer %d fired after %v, expected between %v and %v", i, timeout, min, max)
		}
	}
}

func TestElectionTimer_ConcurrentResets(t *testing.T) {
	min := 50 * time.Millisecond
	max := 100 * time.Millisecond

	timer := NewElectionTimer(min, max)
	timer.Start()
	defer timer.Stop()

	// Reset from multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				time.Sleep(5 * time.Millisecond)
				timer.Reset()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Timer should eventually fire
	select {
	case <-timer.C():
		// Success
	case <-time.After(max + 200*time.Millisecond):
		t.Error("Timer did not fire after concurrent resets")
	}
}

func TestHeartbeatTimer_Ticks(t *testing.T) {
	interval := 20 * time.Millisecond

	timer := NewHeartbeatTimer(interval)
	timer.Start()
	defer timer.Stop()

	// Expect at least 3 ticks
	tickCount := 0
	timeout := time.After(100 * time.Millisecond)

	for tickCount < 3 {
		select {
		case <-timer.C():
			tickCount++
		case <-timeout:
			t.Errorf("Expected at least 3 ticks, got %d", tickCount)
			return
		}
	}

	if tickCount < 3 {
		t.Errorf("Expected at least 3 ticks, got %d", tickCount)
	}
}

func TestHeartbeatTimer_Stop(t *testing.T) {
	interval := 20 * time.Millisecond

	timer := NewHeartbeatTimer(interval)
	timer.Start()

	// Let it tick once
	<-timer.C()

	// Stop the timer
	timer.Stop()

	// Should not receive more ticks (give it some time to be sure)
	select {
	case <-timer.C():
		t.Error("Received tick after timer was stopped")
	case <-time.After(100 * time.Millisecond):
		// Expected
	}
}

func TestHeartbeatTimer_Interval(t *testing.T) {
	interval := 30 * time.Millisecond

	timer := NewHeartbeatTimer(interval)
	timer.Start()
	defer timer.Stop()

	// Measure time between ticks
	<-timer.C() // First tick
	start := time.Now()
	<-timer.C() // Second tick
	elapsed := time.Since(start)

	// Allow some tolerance for scheduling
	tolerance := 10 * time.Millisecond
	if elapsed < interval-tolerance || elapsed > interval+tolerance {
		t.Errorf("Tick interval was %v, expected %v (±%v)", elapsed, interval, tolerance)
	}
}
