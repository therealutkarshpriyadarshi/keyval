package raft

import (
	"sync"
	"testing"
)

func TestNodeState_String(t *testing.T) {
	tests := []struct {
		state    NodeState
		expected string
	}{
		{Follower, "Follower"},
		{Candidate, "Candidate"},
		{Leader, "Leader"},
		{NodeState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("NodeState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestServerState_Concurrent(t *testing.T) {
	ss := NewServerState()

	// Test concurrent reads and writes
	var wg sync.WaitGroup
	const numGoroutines = 100

	// Concurrent term updates
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(term uint64) {
			defer wg.Done()
			ss.SetCurrentTerm(term)
			_ = ss.GetCurrentTerm()
		}(uint64(i))
	}
	wg.Wait()

	// Final term should be set to some value
	if term := ss.GetCurrentTerm(); term >= numGoroutines {
		t.Errorf("Unexpected term value: %d", term)
	}
}

func TestServerState_VotedFor(t *testing.T) {
	ss := NewServerState()

	// Initially empty
	if got := ss.GetVotedFor(); got != "" {
		t.Errorf("Initial VotedFor = %v, want empty string", got)
	}

	// Set and get
	candidateID := "candidate-1"
	ss.SetVotedFor(candidateID)

	if got := ss.GetVotedFor(); got != candidateID {
		t.Errorf("GetVotedFor() = %v, want %v", got, candidateID)
	}
}

func TestServerState_LastLogIndexAndTerm(t *testing.T) {
	ss := NewServerState()

	// Empty log
	idx, term := ss.LastLogIndexAndTerm()
	if idx != 0 || term != 0 {
		t.Errorf("Empty log: got (%d, %d), want (0, 0)", idx, term)
	}

	// Add entries
	ss.AppendEntry(LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal})
	ss.AppendEntry(LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal})
	ss.AppendEntry(LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal})

	idx, term = ss.LastLogIndexAndTerm()
	if idx != 3 || term != 2 {
		t.Errorf("Last log: got (%d, %d), want (3, 2)", idx, term)
	}
}

func TestVolatileState_CommitIndex(t *testing.T) {
	vs := NewVolatileState()

	// Initial value
	if got := vs.GetCommitIndex(); got != 0 {
		t.Errorf("Initial CommitIndex = %d, want 0", got)
	}

	// Set and get
	vs.SetCommitIndex(10)
	if got := vs.GetCommitIndex(); got != 10 {
		t.Errorf("GetCommitIndex() = %d, want 10", got)
	}
}

func TestVolatileState_LastApplied(t *testing.T) {
	vs := NewVolatileState()

	// Initial value
	if got := vs.GetLastApplied(); got != 0 {
		t.Errorf("Initial LastApplied = %d, want 0", got)
	}

	// Set and get
	vs.SetLastApplied(5)
	if got := vs.GetLastApplied(); got != 5 {
		t.Errorf("GetLastApplied() = %d, want 5", got)
	}
}

func TestLeaderState_NextIndex(t *testing.T) {
	peers := []string{"peer1", "peer2", "peer3"}
	ls := NewLeaderState(peers, 10)

	// Check initial values
	for _, peer := range peers {
		if got := ls.GetNextIndex(peer); got != 11 {
			t.Errorf("Initial NextIndex for %s = %d, want 11", peer, got)
		}
	}

	// Update and check
	ls.SetNextIndex("peer1", 15)
	if got := ls.GetNextIndex("peer1"); got != 15 {
		t.Errorf("GetNextIndex(peer1) = %d, want 15", got)
	}

	// Other peers should be unchanged
	if got := ls.GetNextIndex("peer2"); got != 11 {
		t.Errorf("GetNextIndex(peer2) = %d, want 11", got)
	}
}

func TestLeaderState_MatchIndex(t *testing.T) {
	peers := []string{"peer1", "peer2", "peer3"}
	ls := NewLeaderState(peers, 10)

	// Check initial values
	for _, peer := range peers {
		if got := ls.GetMatchIndex(peer); got != 0 {
			t.Errorf("Initial MatchIndex for %s = %d, want 0", peer, got)
		}
	}

	// Update and check
	ls.SetMatchIndex("peer1", 8)
	if got := ls.GetMatchIndex("peer1"); got != 8 {
		t.Errorf("GetMatchIndex(peer1) = %d, want 8", got)
	}
}

func TestElectionState_AddVote(t *testing.T) {
	es := NewElectionState()

	// Initially no votes
	if got := es.GetVoteCount(); got != 0 {
		t.Errorf("Initial vote count = %d, want 0", got)
	}

	// Add vote
	if added := es.AddVote("peer1"); !added {
		t.Error("AddVote(peer1) should return true on first vote")
	}

	if got := es.GetVoteCount(); got != 1 {
		t.Errorf("Vote count after 1 vote = %d, want 1", got)
	}

	// Try to vote again from same peer
	if added := es.AddVote("peer1"); added {
		t.Error("AddVote(peer1) should return false on duplicate vote")
	}

	if got := es.GetVoteCount(); got != 1 {
		t.Errorf("Vote count should still be 1, got %d", got)
	}

	// Add more votes
	es.AddVote("peer2")
	es.AddVote("peer3")

	if got := es.GetVoteCount(); got != 3 {
		t.Errorf("Vote count = %d, want 3", got)
	}
}

func TestElectionState_Reset(t *testing.T) {
	es := NewElectionState()

	// Add some votes
	es.AddVote("peer1")
	es.AddVote("peer2")

	if got := es.GetVoteCount(); got != 2 {
		t.Errorf("Vote count before reset = %d, want 2", got)
	}

	// Reset
	es.Reset()

	if got := es.GetVoteCount(); got != 0 {
		t.Errorf("Vote count after reset = %d, want 0", got)
	}

	// Should be able to vote again
	if added := es.AddVote("peer1"); !added {
		t.Error("AddVote(peer1) should return true after reset")
	}
}

func TestElectionState_Concurrent(t *testing.T) {
	es := NewElectionState()
	const numVotes = 100

	var wg sync.WaitGroup
	wg.Add(numVotes)

	// Concurrent voting from different peers
	for i := 0; i < numVotes; i++ {
		go func(id int) {
			defer wg.Done()
			es.AddVote(string(rune(id)))
		}(i)
	}

	wg.Wait()

	// Should have received all votes
	if got := es.GetVoteCount(); got != numVotes {
		t.Errorf("Vote count = %d, want %d", got, numVotes)
	}
}

func TestClusterConfig_GetPeers(t *testing.T) {
	self := Peer{ID: "node1", Address: "localhost:8001"}
	peers := []Peer{
		{ID: "node2", Address: "localhost:8002"},
		{ID: "node3", Address: "localhost:8003"},
	}

	config := NewClusterConfig(self, peers)

	// Get peers
	gotPeers := config.GetPeers()

	if len(gotPeers) != len(peers) {
		t.Errorf("GetPeers() returned %d peers, want %d", len(gotPeers), len(peers))
	}

	// Modify returned slice (should not affect internal state)
	gotPeers[0].ID = "modified"

	// Get peers again and verify not modified
	gotPeers2 := config.GetPeers()
	if gotPeers2[0].ID == "modified" {
		t.Error("GetPeers() should return a copy, not the internal slice")
	}
}
