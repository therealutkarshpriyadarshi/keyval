package raft

import (
	"testing"
	"time"
)

func TestNewNode(t *testing.T) {
	peers := []Peer{
		{ID: "node2", Address: "localhost:8002"},
		{ID: "node3", Address: "localhost:8003"},
	}

	node, err := NewNode("node1", "localhost:8001", peers)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	if node.GetID() != "node1" {
		t.Errorf("Node ID = %s, want node1", node.GetID())
	}

	if node.GetState() != Follower {
		t.Errorf("Initial state = %s, want Follower", node.GetState())
	}

	if node.GetCurrentTerm() != 0 {
		t.Errorf("Initial term = %d, want 0", node.GetCurrentTerm())
	}

	// Cleanup
	node.Stop()
}

func TestNode_StateTransitions(t *testing.T) {
	peers := []Peer{
		{ID: "node2", Address: "localhost:8002"},
		{ID: "node3", Address: "localhost:8003"},
	}

	node, err := NewNode("node1", "localhost:8001", peers, WithConfig(FastConfig()))
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	// Initially follower
	if state := node.GetState(); state != Follower {
		t.Errorf("Initial state = %s, want Follower", state)
	}

	// Transition to candidate
	node.mu.Lock()
	node.becomeCandidate()
	node.mu.Unlock()

	state := node.GetState()
	term := node.GetCurrentTerm()

	if state != Candidate {
		t.Errorf("After becomeCandidate, state = %s, want Candidate", state)
	}

	if term != 1 {
		t.Errorf("After becomeCandidate, term = %d, want 1", term)
	}

	// Transition to leader
	node.mu.Lock()
	node.becomeLeader()
	node.mu.Unlock()

	state = node.GetState()

	if state != Leader {
		t.Errorf("After becomeLeader, state = %s, want Leader", state)
	}

	if !node.IsLeader() {
		t.Error("IsLeader() = false, want true")
	}
}

func TestNode_StepDown(t *testing.T) {
	peers := []Peer{
		{ID: "node2", Address: "localhost:8002"},
		{ID: "node3", Address: "localhost:8003"},
	}

	node, err := NewNode("node1", "localhost:8001", peers)
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	// Become candidate
	node.mu.Lock()
	node.becomeCandidate()
	node.mu.Unlock()

	if node.GetState() != Candidate {
		t.Fatal("Node should be candidate")
	}

	// Step down to follower
	node.mu.Lock()
	node.stepDown(5)
	node.mu.Unlock()

	if state := node.GetState(); state != Follower {
		t.Errorf("After stepDown, state = %s, want Follower", state)
	}

	if term := node.GetCurrentTerm(); term != 5 {
		t.Errorf("After stepDown, term = %d, want 5", term)
	}

	if votedFor := node.serverState.GetVotedFor(); votedFor != "" {
		t.Errorf("After stepDown, votedFor = %s, want empty", votedFor)
	}
}

func TestNode_StartStop(t *testing.T) {
	peers := []Peer{
		{ID: "node2", Address: "localhost:9002"},
		{ID: "node3", Address: "localhost:9003"},
	}

	node, err := NewNode("node1", "localhost:9001", peers, WithConfig(FastConfig()))
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Start the node
	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Give it a moment
	time.Sleep(50 * time.Millisecond)

	// Stop the node
	if err := node.Stop(); err != nil {
		t.Fatalf("Failed to stop node: %v", err)
	}

	// Try to stop again (should not error)
	if err := node.Stop(); err != nil {
		t.Errorf("Second stop errored: %v", err)
	}
}

func TestNode_Quorum(t *testing.T) {
	tests := []struct {
		name        string
		numPeers    int
		wantQuorum  int
	}{
		{"3-node cluster", 2, 2},
		{"5-node cluster", 4, 3},
		{"7-node cluster", 6, 4},
		{"1-node cluster", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peers := make([]Peer, tt.numPeers)
			for i := 0; i < tt.numPeers; i++ {
				peers[i] = Peer{
					ID:      string(rune('A' + i)),
					Address: "localhost:8000",
				}
			}

			node, err := NewNode("node1", "localhost:8001", peers)
			if err != nil {
				t.Fatalf("Failed to create node: %v", err)
			}
			defer node.Stop()

			quorum := node.calculateQuorum()
			if quorum != tt.wantQuorum {
				t.Errorf("calculateQuorum() = %d, want %d", quorum, tt.wantQuorum)
			}
		})
	}
}
