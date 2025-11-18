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
	// Validate request
	if req == nil {
		return &raftpb.AppendEntriesResponse{
			Term:    n.GetCurrentTerm(),
			Success: false,
		}, nil
	}

	// Process the AppendEntries request
	resp := n.processAppendEntries(req)

	if len(req.Entries) == 0 {
		n.logger.Printf("[DEBUG] Received heartbeat from leader %s for term %d", req.LeaderId, req.Term)
	} else {
		n.logger.Printf("[DEBUG] Received AppendEntries from leader %s with %d entries", req.LeaderId, len(req.Entries))
	}

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
