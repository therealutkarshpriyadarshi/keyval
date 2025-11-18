package raft

import (
	"sync"
	"time"
)

// NodeState represents the state of a Raft node
type NodeState int

const (
	// Follower is the initial state and the state after receiving valid messages from leaders
	Follower NodeState = iota
	// Candidate is the state when a node is campaigning for leadership
	Candidate
	// Leader is the state when a node has won an election
	Leader
)

// String returns a string representation of the node state
func (s NodeState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// LogEntry represents a single entry in the Raft log
type LogEntry struct {
	// Term when entry was received by leader
	Term uint64

	// Index of entry in the log (1-indexed)
	Index uint64

	// Command to be applied to state machine
	Data []byte

	// Type of entry (normal command, config change, or no-op)
	Type EntryType
}

// EntryType defines the type of log entry
type EntryType int

const (
	// EntryNormal is a normal command
	EntryNormal EntryType = iota
	// EntryConfig is a configuration change
	EntryConfig
	// EntryNoOp is a no-op entry (used by leaders on election)
	EntryNoOp
)

// ServerState represents the persistent state on all servers
// This state must be persisted to stable storage before responding to RPCs
type ServerState struct {
	mu sync.RWMutex

	// CurrentTerm is the latest term server has seen
	// Initialized to 0 on first boot, increases monotonically
	CurrentTerm uint64

	// VotedFor is the candidate ID that received vote in current term
	// nil if none
	VotedFor string

	// Log contains log entries
	// First index is 1
	Log []LogEntry
}

// NewServerState creates a new server state
func NewServerState() *ServerState {
	return &ServerState{
		CurrentTerm: 0,
		VotedFor:    "",
		Log:         make([]LogEntry, 0),
	}
}

// GetCurrentTerm returns the current term (thread-safe)
func (s *ServerState) GetCurrentTerm() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentTerm
}

// SetCurrentTerm sets the current term (thread-safe)
func (s *ServerState) SetCurrentTerm(term uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.CurrentTerm = term
}

// GetVotedFor returns the candidate that received vote in current term (thread-safe)
func (s *ServerState) GetVotedFor() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.VotedFor
}

// SetVotedFor sets the candidate that received vote (thread-safe)
func (s *ServerState) SetVotedFor(candidateID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.VotedFor = candidateID
}

// GetLog returns a copy of the log (thread-safe)
func (s *ServerState) GetLog() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logCopy := make([]LogEntry, len(s.Log))
	copy(logCopy, s.Log)
	return logCopy
}

// LastLogIndexAndTerm returns the index and term of the last log entry
func (s *ServerState) LastLogIndexAndTerm() (uint64, uint64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.Log) == 0 {
		return 0, 0
	}

	lastEntry := s.Log[len(s.Log)-1]
	return lastEntry.Index, lastEntry.Term
}

// AppendEntry appends a new entry to the log
func (s *ServerState) AppendEntry(entry LogEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Log = append(s.Log, entry)
}

// VolatileState represents volatile state on all servers
type VolatileState struct {
	mu sync.RWMutex

	// CommitIndex is the index of highest log entry known to be committed
	// Initialized to 0, increases monotonically
	CommitIndex uint64

	// LastApplied is the index of highest log entry applied to state machine
	// Initialized to 0, increases monotonically
	LastApplied uint64
}

// NewVolatileState creates a new volatile state
func NewVolatileState() *VolatileState {
	return &VolatileState{
		CommitIndex: 0,
		LastApplied: 0,
	}
}

// GetCommitIndex returns the commit index (thread-safe)
func (v *VolatileState) GetCommitIndex() uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.CommitIndex
}

// SetCommitIndex sets the commit index (thread-safe)
func (v *VolatileState) SetCommitIndex(index uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.CommitIndex = index
}

// GetLastApplied returns the last applied index (thread-safe)
func (v *VolatileState) GetLastApplied() uint64 {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.LastApplied
}

// SetLastApplied sets the last applied index (thread-safe)
func (v *VolatileState) SetLastApplied(index uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.LastApplied = index
}

// LeaderState represents volatile state on leaders
// Reinitialized after election
type LeaderState struct {
	mu sync.RWMutex

	// NextIndex for each server, index of the next log entry to send to that server
	// Initialized to leader last log index + 1
	NextIndex map[string]uint64

	// MatchIndex for each server, index of highest log entry known to be replicated
	// Initialized to 0, increases monotonically
	MatchIndex map[string]uint64
}

// NewLeaderState creates a new leader state
func NewLeaderState(peers []string, lastLogIndex uint64) *LeaderState {
	ls := &LeaderState{
		NextIndex:  make(map[string]uint64),
		MatchIndex: make(map[string]uint64),
	}

	// Initialize next index to last log index + 1 for all peers
	// Initialize match index to 0 for all peers
	for _, peer := range peers {
		ls.NextIndex[peer] = lastLogIndex + 1
		ls.MatchIndex[peer] = 0
	}

	return ls
}

// GetNextIndex returns the next index for a peer (thread-safe)
func (ls *LeaderState) GetNextIndex(peer string) uint64 {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.NextIndex[peer]
}

// SetNextIndex sets the next index for a peer (thread-safe)
func (ls *LeaderState) SetNextIndex(peer string, index uint64) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.NextIndex[peer] = index
}

// GetMatchIndex returns the match index for a peer (thread-safe)
func (ls *LeaderState) GetMatchIndex(peer string) uint64 {
	ls.mu.RLock()
	defer ls.mu.RUnlock()
	return ls.MatchIndex[peer]
}

// SetMatchIndex sets the match index for a peer (thread-safe)
func (ls *LeaderState) SetMatchIndex(peer string, index uint64) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.MatchIndex[peer] = index
}

// Peer represents a peer in the Raft cluster
type Peer struct {
	// ID is the unique identifier for the peer
	ID string

	// Address is the network address of the peer (host:port)
	Address string
}

// ClusterConfig represents the cluster configuration
type ClusterConfig struct {
	mu sync.RWMutex

	// Peers is the list of all peers in the cluster (excluding self)
	Peers []Peer

	// Self is the current node's peer information
	Self Peer
}

// NewClusterConfig creates a new cluster configuration
func NewClusterConfig(self Peer, peers []Peer) *ClusterConfig {
	return &ClusterConfig{
		Self:  self,
		Peers: peers,
	}
}

// GetPeers returns a copy of the peers list (thread-safe)
func (c *ClusterConfig) GetPeers() []Peer {
	c.mu.RLock()
	defer c.mu.RUnlock()

	peersCopy := make([]Peer, len(c.Peers))
	copy(peersCopy, c.Peers)
	return peersCopy
}

// ElectionState represents the state during an election
type ElectionState struct {
	mu sync.RWMutex

	// VotesReceived tracks which peers have voted for us
	VotesReceived map[string]bool

	// VoteCount is the number of votes received
	VoteCount int

	// ElectionStart is when the election started
	ElectionStart time.Time
}

// NewElectionState creates a new election state
func NewElectionState() *ElectionState {
	return &ElectionState{
		VotesReceived: make(map[string]bool),
		VoteCount:     0,
		ElectionStart: time.Now(),
	}
}

// AddVote records a vote from a peer
func (e *ElectionState) AddVote(peerID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.VotesReceived[peerID] {
		return false // Already voted
	}

	e.VotesReceived[peerID] = true
	e.VoteCount++
	return true
}

// GetVoteCount returns the current vote count
func (e *ElectionState) GetVoteCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.VoteCount
}

// Reset resets the election state
func (e *ElectionState) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.VotesReceived = make(map[string]bool)
	e.VoteCount = 0
	e.ElectionStart = time.Now()
}
