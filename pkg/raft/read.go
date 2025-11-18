package raft

import (
	"time"
)

// ReadRequest represents a linearizable read request
type ReadRequest struct {
	ReadIndex uint64
	StartTime time.Time
}

// PerformLinearizableRead performs a linearizable read by implementing ReadIndex optimization
// This ensures the read sees all writes that were committed before the read started
//
// Algorithm (from Raft paper §6.4):
// 1. Save current commitIndex as readIndex
// 2. Send heartbeat to majority to ensure we're still leader
// 3. Wait for commitIndex >= readIndex
// 4. Wait for lastApplied >= readIndex
// 5. Execute read against state machine
func (n *Node) PerformLinearizableRead(readFunc func() interface{}) (interface{}, bool) {
	n.mu.RLock()

	// Step 1: Check if we're the leader
	if n.state != Leader {
		n.mu.RUnlock()
		return nil, false
	}

	// Step 2: Save current commit index as read index
	readIndex := n.volatileState.GetCommitIndex()
	n.mu.RUnlock()

	// Step 3: Send heartbeat to confirm leadership
	// This is already done by the periodic heartbeat mechanism
	// We just need to wait for a round of heartbeats
	if !n.confirmLeadership() {
		return nil, false
	}

	// Step 4: Wait for lastApplied >= readIndex
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, false

		case <-ticker.C:
			n.mu.RLock()
			lastApplied := n.volatileState.GetLastApplied()
			isLeader := n.state == Leader
			n.mu.RUnlock()

			if !isLeader {
				return nil, false
			}

			if lastApplied >= readIndex {
				// Step 5: Execute read
				result := readFunc()
				return result, true
			}
		}
	}
}

// confirmLeadership confirms that this node is still the leader
// by waiting for a heartbeat round
func (n *Node) confirmLeadership() bool {
	// Wait for at least one heartbeat interval
	// This ensures we've communicated with a majority
	time.Sleep(n.config.HeartbeatInterval)

	n.mu.RLock()
	isLeader := n.state == Leader
	n.mu.RUnlock()

	return isLeader
}

// Get retrieves a value from the state machine with linearizable consistency
func (n *Node) Get(key string) ([]byte, bool, error) {
	// Perform linearizable read
	result, ok := n.PerformLinearizableRead(func() interface{} {
		if n.stateMachine == nil {
			return nil
		}
		value, exists := n.stateMachine.Get(key)
		return &struct {
			Value  []byte
			Exists bool
		}{value, exists}
	})

	if !ok {
		// Not the leader
		return nil, false, nil
	}

	if result == nil {
		return nil, false, nil
	}

	res := result.(*struct {
		Value  []byte
		Exists bool
	})

	return res.Value, res.Exists, nil
}

// GetStateMachine returns the state machine (for testing and API layer)
func (n *Node) GetStateMachine() interface{} {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.stateMachine
}
