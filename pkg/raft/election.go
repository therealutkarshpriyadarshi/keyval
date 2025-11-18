package raft

import (
	"context"
	"sync"

	"github.com/therealutkarshpriyadarshi/keyval/proto"
)

// startElection initiates a new election
func (n *Node) startElection() {
	n.mu.Lock()

	// Become candidate
	n.becomeCandidate()

	currentTerm := n.serverState.GetCurrentTerm()
	lastLogIndex, lastLogTerm := n.serverState.LastLogIndexAndTerm()
	peers := n.cluster.GetPeers()

	n.mu.Unlock()

	n.logger.Printf("[INFO] Node %s starting election for term %d", n.id, currentTerm)

	// Send RequestVote RPCs to all peers in parallel
	var wg sync.WaitGroup
	voteCh := make(chan bool, len(peers))

	for _, peer := range peers {
		wg.Add(1)
		go func(p Peer) {
			defer wg.Done()

			granted := n.sendRequestVote(p, currentTerm, lastLogIndex, lastLogTerm)
			voteCh <- granted
		}(peer)
	}

	// Wait for all RPC calls to complete in a goroutine
	go func() {
		wg.Wait()
		close(voteCh)
	}()

	// Count votes
	votesNeeded := (len(peers) + 1 + 1) / 2 // +1 for self, then majority

	n.mu.RLock()
	votesReceived := n.electionState.GetVoteCount()
	n.mu.RUnlock()

	for vote := range voteCh {
		if vote {
			n.mu.Lock()
			n.electionState.AddVote("")
			votesReceived = n.electionState.GetVoteCount()
			currentState := n.state
			electionTerm := n.serverState.GetCurrentTerm()
			n.mu.Unlock()

			// Check if we've won the election
			if votesReceived >= votesNeeded && currentState == Candidate && electionTerm == currentTerm {
				n.mu.Lock()
				// Double-check we're still a candidate in the same term
				if n.state == Candidate && n.serverState.GetCurrentTerm() == currentTerm {
					n.becomeLeader()
				}
				n.mu.Unlock()
				return
			}
		}
	}

	n.logger.Printf("[INFO] Node %s election for term %d completed with %d votes (needed %d)",
		n.id, currentTerm, votesReceived, votesNeeded)
}

// sendRequestVote sends a RequestVote RPC to a peer
func (n *Node) sendRequestVote(peer Peer, term uint64, lastLogIndex uint64, lastLogTerm uint64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), n.config.RPCTimeout)
	defer cancel()

	client, err := n.clientPool.GetClient(peer.Address)
	if err != nil {
		n.logger.Printf("[ERROR] Failed to get client for %s: %v", peer.Address, err)
		return false
	}

	req := &raftpb.RequestVoteRequest{
		Term:         term,
		CandidateId:  n.id,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	resp, err := client.RequestVote(ctx, req)
	if err != nil {
		n.logger.Printf("[ERROR] RequestVote RPC to %s failed: %v", peer.Address, err)
		return false
	}

	// Handle response
	n.mu.Lock()
	defer n.mu.Unlock()

	// If we discover a higher term, step down
	if resp.Term > n.serverState.GetCurrentTerm() {
		n.stepDown(resp.Term)
		return false
	}

	// If the response is for an old term, ignore it
	if resp.Term < term {
		return false
	}

	return resp.VoteGranted
}

// requestVote handles receiving a RequestVote RPC
// This is called by the RPC handler
func (n *Node) requestVote(req *raftpb.RequestVoteRequest) *raftpb.RequestVoteResponse {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.GetCurrentTerm()
	resp := &raftpb.RequestVoteResponse{
		Term:        currentTerm,
		VoteGranted: false,
	}

	// If request term is less than current term, reject
	if req.Term < currentTerm {
		n.logger.Printf("[DEBUG] Rejecting vote for %s: request term %d < current term %d",
			req.CandidateId, req.Term, currentTerm)
		return resp
	}

	// If request term is greater, update our term and step down
	if req.Term > currentTerm {
		n.stepDown(req.Term)
		currentTerm = req.Term
		resp.Term = currentTerm
	}

	// Check if we've already voted in this term
	votedFor := n.serverState.GetVotedFor()
	if votedFor != "" && votedFor != req.CandidateId {
		n.logger.Printf("[DEBUG] Rejecting vote for %s: already voted for %s in term %d",
			req.CandidateId, votedFor, currentTerm)
		return resp
	}

	// Check if candidate's log is at least as up-to-date as ours
	lastLogIndex, lastLogTerm := n.serverState.LastLogIndexAndTerm()

	logIsUpToDate := false
	if req.LastLogTerm > lastLogTerm {
		// Candidate's last log entry has a higher term
		logIsUpToDate = true
	} else if req.LastLogTerm == lastLogTerm && req.LastLogIndex >= lastLogIndex {
		// Same term, but candidate's log is at least as long
		logIsUpToDate = true
	}

	if !logIsUpToDate {
		n.logger.Printf("[DEBUG] Rejecting vote for %s: log not up-to-date (candidate: idx=%d term=%d, ours: idx=%d term=%d)",
			req.CandidateId, req.LastLogIndex, req.LastLogTerm, lastLogIndex, lastLogTerm)
		return resp
	}

	// Grant vote
	n.serverState.SetVotedFor(req.CandidateId)
	resp.VoteGranted = true

	// Reset election timer since we granted a vote
	n.electionTimer.Reset()

	n.logger.Printf("[INFO] Granted vote to %s for term %d", req.CandidateId, currentTerm)
	return resp
}

// calculateQuorum returns the quorum size for the cluster
func (n *Node) calculateQuorum() int {
	peers := n.cluster.GetPeers()
	clusterSize := len(peers) + 1 // +1 for self
	return (clusterSize / 2) + 1
}

// hasQuorum checks if we have received votes from a quorum
func (n *Node) hasQuorum(voteCount int) bool {
	return voteCount >= n.calculateQuorum()
}
