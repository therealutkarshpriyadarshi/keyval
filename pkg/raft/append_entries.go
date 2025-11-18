package raft

import (
	"context"

	"github.com/therealutkarshpriyadarshi/keyval/proto"
)

// buildAppendEntriesRequest builds an AppendEntries RPC request for a specific peer
// This is used by the leader to replicate log entries or send heartbeats
func (n *Node) buildAppendEntriesRequest(peerID string) *raftpb.AppendEntriesRequest {
	n.mu.RLock()
	defer n.mu.RUnlock()

	currentTerm := n.serverState.GetCurrentTerm()
	commitIndex := n.volatileState.GetCommitIndex()

	// Get the next index to send to this peer
	var nextIndex uint64 = 1
	if n.leaderState != nil {
		nextIndex = n.leaderState.GetNextIndex(peerID)
	}

	// Calculate prevLogIndex and prevLogTerm
	prevLogIndex := nextIndex - 1
	var prevLogTerm uint64 = 0

	if prevLogIndex > 0 {
		prevLogTerm = n.serverState.GetTermAtIndex(prevLogIndex)
	}

	// Get entries to send (from nextIndex to end of log)
	var entries []*raftpb.LogEntry
	lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()

	if nextIndex <= lastLogIndex {
		// We have entries to send
		logEntries := n.serverState.GetEntriesFrom(nextIndex)

		// Limit number of entries per request
		maxEntries := n.config.MaxEntriesPerAppend
		if len(logEntries) > maxEntries {
			logEntries = logEntries[:maxEntries]
		}

		// Convert to protobuf format
		entries = ConvertToProto(logEntries)
	}

	return &raftpb.AppendEntriesRequest{
		Term:         currentTerm,
		LeaderId:     n.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: commitIndex,
	}
}

// GetEntriesFrom returns log entries starting from the given index
func (s *ServerState) GetEntriesFrom(startIndex uint64) []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if startIndex == 0 || startIndex > uint64(len(s.Log)) {
		return []LogEntry{}
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(startIndex) - 1

	// Make a copy to avoid external modification
	result := make([]LogEntry, len(s.Log)-arrayIndex)
	copy(result, s.Log[arrayIndex:])

	return result
}

// sendAppendEntries sends an AppendEntries RPC to a peer
// Returns true if the peer's log was successfully updated
func (n *Node) sendAppendEntries(peer Peer) bool {
	req := n.buildAppendEntriesRequest(peer.ID)

	ctx, cancel := context.WithTimeout(context.Background(), n.config.RPCTimeout)
	defer cancel()

	client, err := n.clientPool.GetClient(peer.Address)
	if err != nil {
		n.logger.Printf("[ERROR] Failed to get client for %s: %v", peer.Address, err)
		return false
	}

	resp, err := client.AppendEntries(ctx, req)
	if err != nil {
		n.logger.Printf("[ERROR] AppendEntries RPC to %s failed: %v", peer.Address, err)
		return false
	}

	// Handle response
	return n.handleAppendEntriesResponse(peer, req, resp)
}

// handleAppendEntriesResponse processes the response from an AppendEntries RPC
func (n *Node) handleAppendEntriesResponse(peer Peer, req *raftpb.AppendEntriesRequest, resp *raftpb.AppendEntriesResponse) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	// If we discover a higher term, step down
	if resp.Term > n.serverState.GetCurrentTerm() {
		n.logger.Printf("[INFO] Discovered higher term %d from %s, stepping down", resp.Term, peer.ID)
		n.stepDown(resp.Term)
		return false
	}

	// If we're no longer leader or the term changed, ignore response
	if n.state != Leader || n.serverState.GetCurrentTerm() != req.Term {
		return false
	}

	if n.leaderState == nil {
		return false
	}

	if resp.Success {
		// Successfully replicated entries to this peer
		if len(req.Entries) > 0 {
			// Update nextIndex and matchIndex
			lastSentIndex := req.Entries[len(req.Entries)-1].Index
			n.leaderState.SetNextIndex(peer.ID, lastSentIndex+1)
			n.leaderState.SetMatchIndex(peer.ID, lastSentIndex)

			n.logger.Printf("[DEBUG] Successfully replicated to %s up to index %d", peer.ID, lastSentIndex)

			// Try to advance commit index
			n.tryAdvanceCommitIndex()
		} else {
			// Heartbeat was successful
			// Ensure nextIndex is at least at the current log end
			currentNextIndex := n.leaderState.GetNextIndex(peer.ID)
			lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
			if currentNextIndex <= lastLogIndex {
				// We have entries to send, will be sent on next replication round
			}
		}
		return true
	}

	// AppendEntries failed - log doesn't match
	// Decrement nextIndex and retry
	nextIndex := n.leaderState.GetNextIndex(peer.ID)

	// Use conflict optimization if available
	if resp.ConflictIndex > 0 {
		// Fast backup using conflict information
		n.leaderState.SetNextIndex(peer.ID, resp.ConflictIndex)
		n.logger.Printf("[DEBUG] Log mismatch with %s, using conflict index %d", peer.ID, resp.ConflictIndex)
	} else {
		// Simple decrement
		if nextIndex > 1 {
			n.leaderState.SetNextIndex(peer.ID, nextIndex-1)
		}
		n.logger.Printf("[DEBUG] Log mismatch with %s, decremented nextIndex to %d", peer.ID, nextIndex-1)
	}

	return false
}

// processAppendEntries processes an incoming AppendEntries RPC
// This is called by followers when they receive AppendEntries from the leader
func (n *Node) processAppendEntries(req *raftpb.AppendEntriesRequest) *raftpb.AppendEntriesResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.GetCurrentTerm()
	resp := &raftpb.AppendEntriesResponse{
		Term:    currentTerm,
		Success: false,
	}

	// 1. Reply false if term < currentTerm (§5.1)
	if req.Term < currentTerm {
		n.logger.Printf("[DEBUG] Rejecting AppendEntries from %s: term %d < current term %d",
			req.LeaderId, req.Term, currentTerm)
		return resp
	}

	// 2. If term >= currentTerm, update term and step down if necessary
	if req.Term >= currentTerm {
		if req.Term > currentTerm {
			n.stepDown(req.Term)
			currentTerm = req.Term
			resp.Term = currentTerm
		}

		// If we're a candidate, step down to follower
		if n.state == Candidate {
			n.setState(Follower)
		}

		// Reset election timer since we received a valid message from leader
		n.electionTimer.Reset()
	}

	// 3. Reply false if log doesn't contain an entry at prevLogIndex
	//    whose term matches prevLogTerm (§5.3)
	if req.PrevLogIndex > 0 {
		// Check if we have an entry at prevLogIndex
		lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()

		if req.PrevLogIndex > lastLogIndex {
			// We don't have the previous entry
			n.logger.Printf("[DEBUG] Log too short: prevLogIndex=%d, our lastIndex=%d",
				req.PrevLogIndex, lastLogIndex)
			resp.ConflictIndex = lastLogIndex + 1
			return resp
		}

		// Check if the term matches
		termAtPrevIndex := n.serverState.GetTermAtIndex(req.PrevLogIndex)
		if termAtPrevIndex != req.PrevLogTerm {
			// Term mismatch
			n.logger.Printf("[DEBUG] Log term mismatch at index %d: have term %d, leader has %d",
				req.PrevLogIndex, termAtPrevIndex, req.PrevLogTerm)

			// Find the first index of the conflicting term
			conflictIndex := req.PrevLogIndex
			for conflictIndex > 0 && n.serverState.GetTermAtIndex(conflictIndex) == termAtPrevIndex {
				conflictIndex--
			}
			resp.ConflictIndex = conflictIndex + 1
			resp.ConflictTerm = termAtPrevIndex

			return resp
		}
	}

	// 4. If an existing entry conflicts with a new one (same index but different terms),
	//    delete the existing entry and all that follow it (§5.3)
	// 5. Append any new entries not already in the log
	if len(req.Entries) > 0 {
		n.applyLogEntries(req.Entries)
	}

	// 6. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
	if req.LeaderCommit > n.volatileState.GetCommitIndex() {
		lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
		newCommitIndex := min(req.LeaderCommit, lastLogIndex)
		n.volatileState.SetCommitIndex(newCommitIndex)

		n.logger.Printf("[DEBUG] Updated commitIndex to %d", newCommitIndex)

		// TODO: Apply committed entries to state machine (Week 4)
	}

	resp.Success = true
	return resp
}

// applyLogEntries applies new log entries, handling conflicts
func (n *Node) applyLogEntries(entries []*raftpb.LogEntry) {
	for _, entry := range entries {
		// Check if we already have this entry
		if entry.Index <= uint64(len(n.serverState.Log)) {
			// We have an entry at this index
			existingTerm := n.serverState.GetTermAtIndex(entry.Index)

			if existingTerm != entry.Term {
				// Conflict! Delete this entry and all following entries
				n.logger.Printf("[INFO] Log conflict at index %d: deleting entries from %d onwards",
					entry.Index, entry.Index)

				// Truncate log
				n.serverState.TruncateLogAfter(entry.Index - 1)

				// Append the new entry
				internalEntry := LogEntry{
					Term:  entry.Term,
					Index: entry.Index,
					Type:  ConvertEntryTypeFromProto(entry.Type),
					Data:  entry.Data,
				}
				n.serverState.AppendEntry(internalEntry)
			}
			// else: entry matches, skip
		} else {
			// New entry beyond our current log
			internalEntry := LogEntry{
				Term:  entry.Term,
				Index: entry.Index,
				Type:  ConvertEntryTypeFromProto(entry.Type),
				Data:  entry.Data,
			}
			n.serverState.AppendEntry(internalEntry)
			n.logger.Printf("[DEBUG] Appended entry at index %d, term %d", entry.Index, entry.Term)
		}
	}
}

// TruncateLogAfter removes all entries after the given index
func (s *ServerState) TruncateLogAfter(index uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index == 0 {
		s.Log = make([]LogEntry, 0)
		return
	}

	if index >= uint64(len(s.Log)) {
		return
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index)
	s.Log = s.Log[:arrayIndex]
}

// min returns the minimum of two uint64 values
func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two uint64 values
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
