package raft

import (
	"context"

	"github.com/therealutkarshpriyadarshi/keyval/proto"
)

// HandleRequestVote handles RequestVote RPC
// This is called by the RPC server when a RequestVote RPC is received
func (n *Node) HandleRequestVote(ctx context.Context, req *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error) {
	n.logger.Printf("[DEBUG] Received RequestVote from %s for term %d", req.CandidateId, req.Term)

	// Validate request
	if req == nil {
		return &raftpb.RequestVoteResponse{
			Term:        n.GetCurrentTerm(),
			VoteGranted: false,
		}, nil
	}

	// Process the vote request
	resp := n.requestVote(req)

	return resp, nil
}

// HandleAppendEntries handles AppendEntries RPC
// This is called by the RPC server when an AppendEntries RPC is received
func (n *Node) HandleAppendEntries(ctx context.Context, req *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.GetCurrentTerm()
	resp := &raftpb.AppendEntriesResponse{
		Term:    currentTerm,
		Success: false,
	}

	// If request term is less than current term, reject
	if req.Term < currentTerm {
		n.logger.Printf("[DEBUG] Rejecting AppendEntries from %s: request term %d < current term %d",
			req.LeaderId, req.Term, currentTerm)
		return resp, nil
	}

	// If request term is greater than or equal to current term, update term and step down
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

		// Reset election timer since we received a message from the leader
		n.electionTimer.Reset()
	}

	// For now, accept empty heartbeats
	// Full log replication will be implemented in week 3
	if len(req.Entries) == 0 {
		resp.Success = true
		n.logger.Printf("[DEBUG] Received heartbeat from leader %s for term %d", req.LeaderId, req.Term)
		return resp, nil
	}

	// TODO: Implement log replication logic (Week 3)
	// For now, just reject entries
	resp.Success = false
	return resp, nil
}

// HandleInstallSnapshot handles InstallSnapshot RPC
// This is called by the RPC server when an InstallSnapshot RPC is received
func (n *Node) HandleInstallSnapshot(ctx context.Context, req *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.GetCurrentTerm()
	resp := &raftpb.InstallSnapshotResponse{
		Term: currentTerm,
	}

	// If request term is less than current term, reject
	if req.Term < currentTerm {
		return resp, nil
	}

	// If request term is greater, update term
	if req.Term > currentTerm {
		n.stepDown(req.Term)
		resp.Term = req.Term
	}

	// TODO: Implement snapshot installation (Week 7)
	n.logger.Printf("[DEBUG] Received InstallSnapshot from %s for term %d (not implemented yet)",
		req.LeaderId, req.Term)

	return resp, nil
}
