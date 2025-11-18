package raft

import (
	"fmt"
	"sort"
	"time"
)

// tryAdvanceCommitIndex attempts to advance the commit index
// This should be called by the leader after successfully replicating entries
// According to Raft paper §5.3 and §5.4.2:
// - If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N,
//   and log[N].term == currentTerm, then set commitIndex = N
func (n *Node) tryAdvanceCommitIndex() {
	// Must be called with lock held
	if n.state != Leader || n.leaderState == nil {
		return
	}

	currentTerm := n.serverState.GetCurrentTerm()
	currentCommitIndex := n.volatileState.GetCommitIndex()
	lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()

	// If no new entries, nothing to commit
	if lastLogIndex <= currentCommitIndex {
		return
	}

	// Collect match indices from all peers (including self)
	peers := n.cluster.GetPeers()
	matchIndices := make([]uint64, 0, len(peers)+1)

	// Add our own log index (leader has replicated to itself)
	matchIndices = append(matchIndices, lastLogIndex)

	// Add match indices from all followers
	for _, peer := range peers {
		matchIndex := n.leaderState.GetMatchIndex(peer.ID)
		matchIndices = append(matchIndices, matchIndex)
	}

	// Sort match indices in descending order
	sort.Slice(matchIndices, func(i, j int) bool {
		return matchIndices[i] > matchIndices[j]
	})

	// Find the highest index that has been replicated to a majority
	// The majority index is at position quorumIndex in the sorted array
	clusterSize := len(peers) + 1 // +1 for self
	quorumIndex := (clusterSize - 1) / 2

	if quorumIndex >= len(matchIndices) {
		// This shouldn't happen, but be safe
		return
	}

	majorityMatchIndex := matchIndices[quorumIndex]

	// Only commit entries from the current term (§5.4.2)
	// This prevents committing entries from previous terms directly
	for N := majorityMatchIndex; N > currentCommitIndex; N-- {
		termAtN := n.serverState.GetTermAtIndex(N)

		if termAtN == currentTerm {
			// We can commit up to N
			n.volatileState.SetCommitIndex(N)
			n.logger.Printf("[INFO] Advanced commitIndex to %d (majority match: %d)",
				N, majorityMatchIndex)

			// TODO: Trigger application of committed entries to state machine (Week 4)
			// For now, just update the index
			return
		}
	}
}

// GetCommitIndex returns the current commit index
func (n *Node) GetCommitIndex() uint64 {
	return n.volatileState.GetCommitIndex()
}

// GetLastApplied returns the last applied index
func (n *Node) GetLastApplied() uint64 {
	return n.volatileState.GetLastApplied()
}

// waitForCommit waits for a specific log index to be committed
// This is useful for clients waiting for their operation to be committed
// Returns true if the index was committed, false if timeout or node is no longer leader
func (n *Node) waitForCommit(index uint64, timeout <-chan struct{}) bool {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return false
		case <-ticker.C:
			n.mu.RLock()
			commitIndex := n.volatileState.GetCommitIndex()
			isLeader := n.state == Leader
			n.mu.RUnlock()

			if !isLeader {
				return false
			}

			if commitIndex >= index {
				return true
			}
		}
	}
}

// getQuorumSize returns the number of nodes needed for a quorum
func (n *Node) getQuorumSize() int {
	peers := n.cluster.GetPeers()
	clusterSize := len(peers) + 1 // +1 for self
	return (clusterSize / 2) + 1
}

// isCommitted checks if a specific log index has been committed
func (n *Node) isCommitted(index uint64) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return index <= n.volatileState.GetCommitIndex()
}

// GetCommittedEntries returns all log entries that have been committed but not yet applied
func (n *Node) GetCommittedEntries() []LogEntry {
	n.mu.RLock()
	defer n.mu.RUnlock()

	commitIndex := n.volatileState.GetCommitIndex()
	lastApplied := n.volatileState.GetLastApplied()

	if lastApplied >= commitIndex {
		return []LogEntry{}
	}

	entries := make([]LogEntry, 0)
	for i := lastApplied + 1; i <= commitIndex; i++ {
		if entry, err := n.getLogEntry(i); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries
}

// getLogEntry retrieves a log entry by index (must be called with lock held)
func (n *Node) getLogEntry(index uint64) (LogEntry, error) {
	if index == 0 || index > uint64(len(n.serverState.Log)) {
		return LogEntry{}, fmt.Errorf("index %d out of bounds", index)
	}

	arrayIndex := int(index) - 1
	return n.serverState.Log[arrayIndex], nil
}

// applyCommittedEntries applies committed entries to the state machine
// This should be called periodically by a separate goroutine
func (n *Node) applyCommittedEntries() {
	n.mu.Lock()
	defer n.mu.Unlock()

	commitIndex := n.volatileState.GetCommitIndex()
	lastApplied := n.volatileState.GetLastApplied()

	if lastApplied >= commitIndex {
		return
	}

	// Apply entries from lastApplied+1 to commitIndex
	for i := lastApplied + 1; i <= commitIndex; i++ {
		_, err := n.getLogEntry(i)
		if err != nil {
			n.logger.Printf("[ERROR] Failed to get log entry %d: %v", i, err)
			break
		}

		// TODO: Apply to state machine (Week 4)
		// For now, just log that we would apply it
		n.logger.Printf("[DEBUG] Would apply entry %d to state machine (not implemented yet)", i)

		// Update last applied
		n.volatileState.SetLastApplied(i)
	}
}

// startApplyLoop starts a goroutine that periodically applies committed entries
func (n *Node) startApplyLoop() {
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-n.stopCh:
				return
			case <-ticker.C:
				n.applyCommittedEntries()
			}
		}
	}()
}
