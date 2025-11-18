package raft

import (
	"sync"
	"time"
)

// startReplication starts the log replication process for leaders
// This goroutine periodically attempts to replicate log entries to all followers
func (n *Node) startReplication() {
	n.logger.Printf("[INFO] Starting log replication")

	// Send initial round of AppendEntries to establish leadership
	n.replicateToAll()

	// Start periodic replication
	ticker := time.NewTicker(n.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.mu.RLock()
			isLeader := n.state == Leader
			n.mu.RUnlock()

			if !isLeader {
				return
			}

			n.replicateToAll()
		}
	}
}

// replicateToAll replicates log entries to all peers
func (n *Node) replicateToAll() {
	n.mu.RLock()

	if n.state != Leader {
		n.mu.RUnlock()
		return
	}

	peers := n.cluster.GetPeers()
	n.mu.RUnlock()

	// Replicate to each peer in parallel
	var wg sync.WaitGroup
	for _, peer := range peers {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()
			n.replicateToPeer(p)
		}(peer)
	}

	wg.Wait()
}

// replicateToPeer attempts to replicate log entries to a specific peer
// This is called by the leader for each follower
func (n *Node) replicateToPeer(peer Peer) {
	// Try multiple times if there are entries to replicate
	maxRetries := n.config.MaxAppendEntriesRetries

	for attempt := 0; attempt < maxRetries; attempt++ {
		n.mu.RLock()
		if n.state != Leader {
			n.mu.RUnlock()
			return
		}

		// Check if we have entries to send
		nextIndex := uint64(1)
		if n.leaderState != nil {
			nextIndex = n.leaderState.GetNextIndex(peer.ID)
		}
		lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
		hasEntriesToSend := nextIndex <= lastLogIndex
		n.mu.RUnlock()

		// Send AppendEntries (or heartbeat if no entries)
		success := n.sendAppendEntries(peer)

		if success {
			// Successfully replicated
			if hasEntriesToSend {
				// We sent actual entries, check if there are more
				n.mu.RLock()
				newNextIndex := uint64(1)
				if n.leaderState != nil {
					newNextIndex = n.leaderState.GetNextIndex(peer.ID)
				}
				stillHasMore := newNextIndex <= lastLogIndex
				n.mu.RUnlock()

				if stillHasMore {
					// There are more entries to send, continue
					continue
				}
			}
			// No more entries or heartbeat sent successfully
			return
		}

		// Failed to replicate, wait a bit before retrying
		if attempt < maxRetries-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	n.logger.Printf("[WARN] Failed to replicate to %s after %d attempts", peer.ID, maxRetries)
}

// Replicate is a public method to trigger replication
// This can be called when a new entry is appended to the log
func (n *Node) Replicate() {
	n.mu.RLock()
	if n.state != Leader {
		n.mu.RUnlock()
		return
	}
	n.mu.RUnlock()

	go n.replicateToAll()
}

// AppendLogEntry appends a new entry to the leader's log and triggers replication
// This is the main entry point for clients to submit commands
// Returns the index of the new entry and whether this node is the leader
func (n *Node) AppendLogEntry(data []byte, entryType EntryType) (uint64, bool) {
	n.mu.Lock()

	// Only leaders can append entries
	if n.state != Leader {
		n.mu.Unlock()
		return 0, false
	}

	currentTerm := n.serverState.GetCurrentTerm()
	lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
	newIndex := lastLogIndex + 1

	entry := LogEntry{
		Term:  currentTerm,
		Index: newIndex,
		Data:  data,
		Type:  entryType,
	}

	n.serverState.AppendEntry(entry)
	n.logger.Printf("[INFO] Leader appended entry at index %d, term %d", newIndex, currentTerm)

	n.mu.Unlock()

	// Trigger immediate replication
	go n.replicateToAll()

	return newIndex, true
}

// appendNoOpEntry appends a no-op entry to the log
// Leaders append a no-op entry at the start of their term to commit entries from previous terms
func (n *Node) appendNoOpEntry() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.state != Leader {
		return
	}

	currentTerm := n.serverState.GetCurrentTerm()
	lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
	newIndex := lastLogIndex + 1

	entry := LogEntry{
		Term:  currentTerm,
		Index: newIndex,
		Data:  nil,
		Type:  EntryNoOp,
	}

	n.serverState.AppendEntry(entry)
	n.logger.Printf("[INFO] Leader appended no-op entry at index %d, term %d", newIndex, currentTerm)

	// Trigger immediate replication
	go n.replicateToAll()
}

// GetReplicationStatus returns the replication status for all peers
// Returns a map of peer ID to their match index
func (n *Node) GetReplicationStatus() map[string]uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.state != Leader || n.leaderState == nil {
		return nil
	}

	status := make(map[string]uint64)
	peers := n.cluster.GetPeers()

	for _, peer := range peers {
		status[peer.ID] = n.leaderState.GetMatchIndex(peer.ID)
	}

	return status
}

// GetNextIndex returns the next index for a peer
// This is mainly for testing and debugging
func (n *Node) GetNextIndex(peerID string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.leaderState == nil {
		return 0
	}

	return n.leaderState.GetNextIndex(peerID)
}

// GetMatchIndex returns the match index for a peer
// This is mainly for testing and debugging
func (n *Node) GetMatchIndex(peerID string) uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.leaderState == nil {
		return 0
	}

	return n.leaderState.GetMatchIndex(peerID)
}
