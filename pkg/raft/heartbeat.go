package raft

import (
	"context"
	"sync"
	"time"

	raftpb "github.com/therealutkarshpriyadarshi/keyval/proto"
)

// HeartbeatTimer manages the heartbeat timer for leaders
type HeartbeatTimer struct {
	mu       sync.Mutex
	interval time.Duration
	timer    *time.Timer
	c        chan struct{}
	stopCh   chan struct{}
	running  bool
}

// NewHeartbeatTimer creates a new heartbeat timer
func NewHeartbeatTimer(interval time.Duration) *HeartbeatTimer {
	return &HeartbeatTimer{
		interval: interval,
		c:        make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
		running:  false,
	}
}

// Start starts the heartbeat timer
func (ht *HeartbeatTimer) Start() {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if ht.running {
		return
	}

	ht.timer = time.NewTimer(ht.interval)
	ht.running = true

	go ht.run()
}

// Stop stops the heartbeat timer
func (ht *HeartbeatTimer) Stop() {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if !ht.running {
		return
	}

	ht.running = false
	close(ht.stopCh)

	if ht.timer != nil {
		ht.timer.Stop()
	}
}

// Reset resets the heartbeat timer
func (ht *HeartbeatTimer) Reset() {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if !ht.running {
		return
	}

	if ht.timer != nil {
		ht.timer.Reset(ht.interval)
	}
}

// C returns the channel that receives timer events
func (ht *HeartbeatTimer) C() <-chan struct{} {
	return ht.c
}

// run is the main loop for the heartbeat timer
func (ht *HeartbeatTimer) run() {
	for {
		select {
		case <-ht.stopCh:
			return
		case <-ht.timer.C:
			// Send tick
			select {
			case ht.c <- struct{}{}:
			default:
				// Channel full, skip this tick
			}

			// Reset timer for next heartbeat
			ht.mu.Lock()
			if ht.running && ht.timer != nil {
				ht.timer.Reset(ht.interval)
			}
			ht.mu.Unlock()
		}
	}
}

// sendHeartbeats sends heartbeat AppendEntries to all peers
// This should be called by the leader periodically
func (n *Node) sendHeartbeats() {
	n.mu.RLock()

	// Only leaders send heartbeats
	if n.state != Leader {
		n.mu.RUnlock()
		return
	}

	currentTerm := n.serverState.GetCurrentTerm()
	peers := n.cluster.GetPeers()
	commitIndex := n.volatileState.GetCommitIndex()

	n.mu.RUnlock()

	n.logger.Printf("[DEBUG] Leader %s sending heartbeats for term %d", n.id, currentTerm)

	// Send heartbeats to all peers in parallel
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			n.sendHeartbeatToPeer(p, currentTerm, commitIndex)
		}(peer)
	}

	wg.Wait()
}

// sendHeartbeatToPeer sends a heartbeat to a single peer
func (n *Node) sendHeartbeatToPeer(peer Peer, term uint64, leaderCommit uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), n.config.RPCTimeout)
	defer cancel()

	client, err := n.clientPool.GetClient(peer.Address)
	if err != nil {
		n.logger.Printf("[ERROR] Failed to get client for %s: %v", peer.Address, err)
		return
	}

	// Get the previous log index and term for this peer
	n.mu.RLock()
	var prevLogIndex, prevLogTerm uint64
	if n.leaderState != nil {
		nextIndex := n.leaderState.GetNextIndex(peer.ID)
		prevLogIndex = nextIndex - 1

		// Get term of previous log entry
		if prevLogIndex > 0 {
			prevLogTerm = n.serverState.GetTermAtIndex(prevLogIndex)
		}
	}
	n.mu.RUnlock()

	// Create empty AppendEntries request (heartbeat)
	req := &raftpb.AppendEntriesRequest{
		Term:         term,
		LeaderId:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      nil, // Empty for heartbeat
		LeaderCommit: leaderCommit,
	}

	resp, err := client.AppendEntries(ctx, req)
	if err != nil {
		n.logger.Printf("[ERROR] Heartbeat to %s failed: %v", peer.Address, err)
		return
	}

	// Handle response
	n.mu.Lock()
	defer n.mu.Unlock()

	// If we discover a higher term, step down
	if resp.Term > n.serverState.GetCurrentTerm() {
		n.logger.Printf("[INFO] Discovered higher term %d (ours: %d), stepping down",
			resp.Term, n.serverState.GetCurrentTerm())
		n.stepDown(resp.Term)
		return
	}

	// If we're no longer leader or term changed, ignore response
	if n.state != Leader || n.serverState.GetCurrentTerm() != term {
		return
	}

	// Heartbeat was successful
	if resp.Success {
		n.logger.Printf("[DEBUG] Heartbeat to %s successful", peer.Address)
	} else {
		// Heartbeat failed - this could mean log inconsistency
		// In week 3, we'll handle this by decrementing nextIndex and retrying with actual entries
		n.logger.Printf("[DEBUG] Heartbeat to %s failed (log inconsistency)", peer.Address)

		// Decrement nextIndex for this peer to try sending actual log entries next time
		if n.leaderState != nil {
			nextIndex := n.leaderState.GetNextIndex(peer.ID)
			if nextIndex > 1 {
				n.leaderState.SetNextIndex(peer.ID, nextIndex-1)
			}
		}
	}
}

// GetTermAtIndex is a helper method to get the term at a specific index
// This is used by heartbeat to get prevLogTerm
func (s *ServerState) GetTermAtIndex(index uint64) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index == 0 || index > uint64(len(s.Log)) {
		return 0
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index) - 1
	return s.Log[arrayIndex].Term
}

// startHeartbeatTimer starts the heartbeat timer for leaders
// This should be called when a node becomes leader
func (n *Node) startHeartbeatTimer() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.heartbeatTimer == nil {
		n.heartbeatTimer = NewHeartbeatTimer(n.config.HeartbeatInterval)
	}

	n.heartbeatTimer.Start()

	// Start goroutine to handle heartbeat ticks
	go n.handleHeartbeatTicks()
}

// stopHeartbeatTimer stops the heartbeat timer
// This should be called when a node is no longer leader
func (n *Node) stopHeartbeatTimer() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.heartbeatTimer != nil {
		n.heartbeatTimer.Stop()
	}
}

// handleHeartbeatTicks handles heartbeat timer ticks
func (n *Node) handleHeartbeatTicks() {
	for {
		select {
		case <-n.stopCh:
			return
		case <-n.heartbeatTimer.C():
			n.mu.RLock()
			isLeader := n.state == Leader
			n.mu.RUnlock()

			if !isLeader {
				return
			}

			n.sendHeartbeats()
		}
	}
}
