package raft

import (
	"testing"
	"time"
)

func TestLeaseManager_ExtendLease(t *testing.T) {
	leaseDuration := 100 * time.Millisecond
	lm := NewLeaseManager(leaseDuration)

	// Initially no valid lease
	if lm.HasValidLease() {
		t.Error("expected no valid lease initially")
	}

	// Extend lease
	lm.ExtendLease()

	// Should have valid lease
	if !lm.HasValidLease() {
		t.Error("expected valid lease after extension")
	}

	// Lease should expire after duration
	time.Sleep(leaseDuration + 10*time.Millisecond)

	if lm.HasValidLease() {
		t.Error("expected lease to expire")
	}
}

func TestLeaseManager_RevokeLease(t *testing.T) {
	leaseDuration := 100 * time.Millisecond
	lm := NewLeaseManager(leaseDuration)

	// Extend lease
	lm.ExtendLease()

	if !lm.HasValidLease() {
		t.Fatal("expected valid lease")
	}

	// Revoke lease
	lm.RevokeLease()

	// Should no longer have valid lease
	if lm.HasValidLease() {
		t.Error("expected lease to be revoked")
	}
}

func TestLeaseManager_TimeUntilExpiry(t *testing.T) {
	leaseDuration := 100 * time.Millisecond
	lm := NewLeaseManager(leaseDuration)

	// No lease initially
	if timeUntil := lm.TimeUntilExpiry(); timeUntil != 0 {
		t.Errorf("expected 0 duration, got %v", timeUntil)
	}

	// Extend lease
	lm.ExtendLease()

	// Should have time remaining
	timeUntil := lm.TimeUntilExpiry()
	if timeUntil <= 0 || timeUntil > leaseDuration {
		t.Errorf("expected time between 0 and %v, got %v", leaseDuration, timeUntil)
	}

	// Wait for half the duration
	time.Sleep(leaseDuration / 2)

	// Should have roughly half the time remaining
	timeUntil = lm.TimeUntilExpiry()
	if timeUntil > leaseDuration/2+10*time.Millisecond {
		t.Errorf("expected time around half duration, got %v", timeUntil)
	}
}

func TestLeaseManager_GetLastHeartbeatTime(t *testing.T) {
	lm := NewLeaseManager(100 * time.Millisecond)

	// Initially zero time
	if !lm.GetLastHeartbeatTime().IsZero() {
		t.Error("expected zero time initially")
	}

	// Extend lease
	beforeExtend := time.Now()
	lm.ExtendLease()
	afterExtend := time.Now()

	// Last heartbeat time should be between before and after
	lastTime := lm.GetLastHeartbeatTime()
	if lastTime.Before(beforeExtend) || lastTime.After(afterExtend) {
		t.Error("last heartbeat time not in expected range")
	}
}

func TestLeaseManager_MultipleExtensions(t *testing.T) {
	leaseDuration := 50 * time.Millisecond
	lm := NewLeaseManager(leaseDuration)

	// Extend lease multiple times
	for i := 0; i < 5; i++ {
		lm.ExtendLease()
		time.Sleep(20 * time.Millisecond)

		// Should still have valid lease
		if !lm.HasValidLease() {
			t.Errorf("iteration %d: expected valid lease", i)
		}
	}

	// Stop extending and wait for expiration
	time.Sleep(leaseDuration + 10*time.Millisecond)

	if lm.HasValidLease() {
		t.Error("expected lease to expire")
	}
}

func TestLeaseManager_ConcurrentAccess(t *testing.T) {
	lm := NewLeaseManager(100 * time.Millisecond)

	done := make(chan struct{})

	// Concurrent extensions
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				lm.ExtendLease()
				lm.HasValidLease()
				lm.GetLeaseExpiry()
				lm.TimeUntilExpiry()
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should still have valid lease
	if !lm.HasValidLease() {
		t.Error("expected valid lease after concurrent operations")
	}
}

func TestNode_UpdateLeaseOnHeartbeat(t *testing.T) {
	config := DefaultConfig()
	config.HeartbeatInterval = 50 * time.Millisecond
	config.ElectionTimeoutMin = 150 * time.Millisecond
	config.ElectionTimeoutMax = 300 * time.Millisecond

	node := NewNode("node1", []string{"node2", "node3"}, config, nil)

	// Initialize lease manager
	node.leaseManager = NewLeaseManager(80 * time.Millisecond) // Less than election timeout

	// Set node as leader
	node.mu.Lock()
	node.state = Leader
	node.mu.Unlock()

	// Initially no valid lease
	if node.leaseManager.HasValidLease() {
		t.Error("expected no valid lease initially")
	}

	// Update with majority responses (2 out of 3)
	node.UpdateLeaseOnHeartbeat(2, 3)

	// Should have valid lease
	if !node.leaseManager.HasValidLease() {
		t.Error("expected valid lease after majority heartbeat")
	}

	// Update with minority responses (1 out of 3)
	time.Sleep(100 * time.Millisecond) // Wait for lease to expire
	node.UpdateLeaseOnHeartbeat(1, 3)

	// Should not have valid lease
	if node.leaseManager.HasValidLease() {
		t.Error("expected no lease with minority heartbeat")
	}
}

func TestNode_RevokeLeaseOnStepDown(t *testing.T) {
	config := DefaultConfig()
	node := NewNode("node1", []string{"node2", "node3"}, config, nil)

	// Initialize lease manager
	node.leaseManager = NewLeaseManager(100 * time.Millisecond)

	// Set node as leader and extend lease
	node.mu.Lock()
	node.state = Leader
	node.mu.Unlock()

	node.leaseManager.ExtendLease()

	if !node.leaseManager.HasValidLease() {
		t.Fatal("expected valid lease")
	}

	// Revoke lease on step down
	node.RevokeLeaseOnStepDown()

	// Should no longer have valid lease
	if node.leaseManager.HasValidLease() {
		t.Error("expected lease to be revoked after step down")
	}
}
