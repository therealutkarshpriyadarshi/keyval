package raft

import (
	"sync"
	"time"
)

// LeaseManager manages read leases for the leader
// This allows the leader to serve reads without waiting for heartbeats
type LeaseManager struct {
	mu sync.RWMutex

	// Current lease expiration time
	leaseExpiry time.Time

	// Lease duration (should be less than election timeout)
	leaseDuration time.Duration

	// Last time we confirmed leadership via heartbeat
	lastHeartbeatConfirm time.Time
}

// NewLeaseManager creates a new lease manager
func NewLeaseManager(leaseDuration time.Duration) *LeaseManager {
	return &LeaseManager{
		leaseDuration: leaseDuration,
	}
}

// ExtendLease extends the lease based on successful heartbeat responses
// Should be called when a majority of followers acknowledge a heartbeat
func (lm *LeaseManager) ExtendLease() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	now := time.Now()
	lm.leaseExpiry = now.Add(lm.leaseDuration)
	lm.lastHeartbeatConfirm = now
}

// HasValidLease returns true if the current lease is still valid
func (lm *LeaseManager) HasValidLease() bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return time.Now().Before(lm.leaseExpiry)
}

// GetLeaseExpiry returns the current lease expiration time
func (lm *LeaseManager) GetLeaseExpiry() time.Time {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.leaseExpiry
}

// RevokeLease immediately revokes the current lease
// Should be called when stepping down from leader
func (lm *LeaseManager) RevokeLease() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.leaseExpiry = time.Time{}
}

// TimeUntilExpiry returns the duration until lease expiration
func (lm *LeaseManager) TimeUntilExpiry() time.Duration {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	if lm.leaseExpiry.IsZero() {
		return 0
	}

	remaining := time.Until(lm.leaseExpiry)
	if remaining < 0 {
		return 0
	}

	return remaining
}

// GetLastHeartbeatTime returns when we last confirmed leadership
func (lm *LeaseManager) GetLastHeartbeatTime() time.Time {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.lastHeartbeatConfirm
}

// PerformLeaseRead performs a read using the lease mechanism
// This is faster than ReadIndex as it doesn't require waiting for heartbeats
func (n *Node) PerformLeaseRead(readFunc func() interface{}) (interface{}, bool) {
	n.mu.RLock()

	// Check if we're the leader
	if n.state != Leader {
		n.mu.RUnlock()
		return nil, false
	}

	// Check if we have a valid lease
	if n.leaseManager == nil || !n.leaseManager.HasValidLease() {
		n.mu.RUnlock()
		// Fall back to ReadIndex if no valid lease
		return n.PerformLinearizableRead(readFunc)
	}

	n.mu.RUnlock()

	// We have a valid lease, can read immediately
	// Still need to ensure applied state is up to date
	n.mu.RLock()
	commitIndex := n.volatileState.GetCommitIndex()
	lastApplied := n.volatileState.GetLastApplied()
	n.mu.RUnlock()

	// Wait for lastApplied to catch up to commitIndex
	if lastApplied < commitIndex {
		timeout := time.After(1 * time.Second)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				return nil, false
			case <-ticker.C:
				n.mu.RLock()
				lastApplied = n.volatileState.GetLastApplied()
				hasLease := n.leaseManager.HasValidLease()
				isLeader := n.state == Leader
				n.mu.RUnlock()

				if !isLeader || !hasLease {
					return nil, false
				}

				if lastApplied >= commitIndex {
					result := readFunc()
					return result, true
				}
			}
		}
	}

	// Execute read
	result := readFunc()
	return result, true
}

// UpdateLeaseOnHeartbeat should be called when heartbeat responses are received
// count: number of successful responses
// total: total number of peers (including self)
func (n *Node) UpdateLeaseOnHeartbeat(successCount, totalPeers int) {
	n.mu.RLock()
	isLeader := n.state == Leader
	n.mu.RUnlock()

	if !isLeader {
		return
	}

	// Extend lease if we have a majority
	quorum := (totalPeers / 2) + 1
	if successCount >= quorum {
		if n.leaseManager != nil {
			n.leaseManager.ExtendLease()
		}
	}
}

// RevokeLeaseOnStepDown should be called when stepping down from leader
func (n *Node) RevokeLeaseOnStepDown() {
	if n.leaseManager != nil {
		n.leaseManager.RevokeLease()
	}
}
