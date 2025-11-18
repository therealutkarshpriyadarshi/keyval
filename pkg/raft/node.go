package raft

import (
	"fmt"
	"log"
	"sync"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/rpc"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// Node represents a Raft node
type Node struct {
	mu sync.RWMutex

	// Configuration
	config *Config

	// Node identity and cluster info
	id      string
	cluster *ClusterConfig

	// Persistent state on all servers
	serverState *ServerState

	// Volatile state on all servers
	volatileState *VolatileState

	// Volatile state on leaders (reinitialized after election)
	leaderState *LeaderState

	// Current state (Follower, Candidate, or Leader)
	state NodeState

	// Election timer
	electionTimer *ElectionTimer

	// Heartbeat timer (only active for leaders)
	heartbeatTimer *HeartbeatTimer

	// Election state (only valid during elections)
	electionState *ElectionState

	// RPC server and client pool
	rpcServer   *rpc.Server
	clientPool  *rpc.ClientPool

	// State machine
	stateMachine statemachine.StateMachine

	// Channels for communication
	stopCh chan struct{}

	// Logger
	logger *log.Logger
}

// NodeOption is a function that configures a Node
type NodeOption func(*Node)

// WithLogger sets a custom logger for the node
func WithLogger(logger *log.Logger) NodeOption {
	return func(n *Node) {
		n.logger = logger
	}
}

// WithConfig sets a custom configuration for the node
func WithConfig(config *Config) NodeOption {
	return func(n *Node) {
		n.config = config
	}
}

// WithStateMachine sets a custom state machine for the node
func WithStateMachine(sm statemachine.StateMachine) NodeOption {
	return func(n *Node) {
		n.stateMachine = sm
	}
}

// NewNode creates a new Raft node
func NewNode(id string, address string, peers []Peer, options ...NodeOption) (*Node, error) {
	if id == "" {
		return nil, fmt.Errorf("node ID cannot be empty")
	}

	if address == "" {
		return nil, fmt.Errorf("node address cannot be empty")
	}

	n := &Node{
		id: id,
		cluster: NewClusterConfig(Peer{
			ID:      id,
			Address: address,
		}, peers),
		serverState:   NewServerState(),
		volatileState: NewVolatileState(),
		state:         Follower,
		electionState: NewElectionState(),
		config:        DefaultConfig(),
		stopCh:        make(chan struct{}),
		logger:        log.Default(),
		stateMachine:  statemachine.NewKVStateMachine(), // Default state machine
	}

	// Apply options
	for _, opt := range options {
		opt(n)
	}

	// Validate configuration
	if err := n.config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create RPC server
	rpcServer, err := rpc.NewServer(address, n)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC server: %w", err)
	}
	n.rpcServer = rpcServer

	// Create client pool
	n.clientPool = rpc.NewClientPool(n.config.RPCTimeout)

	// Create election timer
	n.electionTimer = NewElectionTimer(
		n.config.ElectionTimeoutMin,
		n.config.ElectionTimeoutMax,
	)

	return n, nil
}

// Start starts the Raft node
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Start RPC server
	if err := n.rpcServer.Start(); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}

	// Start election timer
	n.electionTimer.Start()

	// Start apply loop for committing entries
	n.startApplyLoop()

	// Start main loop
	go n.run()

	n.logger.Printf("[INFO] Node %s started at %s", n.id, n.cluster.Self.Address)
	return nil
}

// Stop stops the Raft node
func (n *Node) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	select {
	case <-n.stopCh:
		return nil // Already stopped
	default:
		close(n.stopCh)
	}

	// Stop election timer
	n.electionTimer.Stop()

	// Stop RPC server
	if err := n.rpcServer.Stop(); err != nil {
		n.logger.Printf("[ERROR] Failed to stop RPC server: %v", err)
	}

	// Close all client connections
	if err := n.clientPool.CloseAll(); err != nil {
		n.logger.Printf("[ERROR] Failed to close client pool: %v", err)
	}

	n.logger.Printf("[INFO] Node %s stopped", n.id)
	return nil
}

// run is the main event loop for the node
func (n *Node) run() {
	for {
		select {
		case <-n.stopCh:
			return
		case <-n.electionTimer.C():
			n.handleElectionTimeout()
		}
	}
}

// handleElectionTimeout handles an election timeout
func (n *Node) handleElectionTimeout() {
	n.mu.Lock()
	state := n.state
	n.mu.Unlock()

	switch state {
	case Follower, Candidate:
		// Start a new election
		n.startElection()
	case Leader:
		// Leaders don't have election timeouts
		// (they send heartbeats instead)
	}
}

// GetState returns the current state of the node
func (n *Node) GetState() NodeState {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state
}

// setState sets the state of the node (must hold lock)
func (n *Node) setState(state NodeState) {
	oldState := n.state
	n.state = state

	if oldState != state {
		n.logger.Printf("[INFO] Node %s transitioned from %s to %s (term %d)",
			n.id, oldState, state, n.serverState.GetCurrentTerm())

		// Reset election timer on state change
		n.electionTimer.Reset()

		// Initialize leader state if becoming leader
		if state == Leader {
			lastLogIndex, _ := n.serverState.LastLogIndexAndTerm()
			peerIDs := make([]string, 0, len(n.cluster.Peers))
			for _, peer := range n.cluster.Peers {
				peerIDs = append(peerIDs, peer.ID)
			}
			n.leaderState = NewLeaderState(peerIDs, lastLogIndex)
		} else {
			n.leaderState = nil
		}
	}
}

// GetCurrentTerm returns the current term
func (n *Node) GetCurrentTerm() uint64 {
	return n.serverState.GetCurrentTerm()
}

// IsLeader returns true if this node is the leader
func (n *Node) IsLeader() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.state == Leader
}

// GetLeaderID returns the current leader ID (empty string if unknown)
func (n *Node) GetLeaderID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.state == Leader {
		return n.id
	}

	// In a real implementation, we would track the leader ID
	// For now, return empty string if not the leader
	return ""
}

// GetID returns the node ID
func (n *Node) GetID() string {
	return n.id
}

// stepDown steps down from candidate or leader to follower
// Must be called with lock held
func (n *Node) stepDown(newTerm uint64) {
	// If we're stepping down from leader, stop heartbeat timer
	if n.state == Leader {
		go n.stopHeartbeatTimer()
	}

	if newTerm > n.serverState.GetCurrentTerm() {
		n.serverState.SetCurrentTerm(newTerm)
		n.serverState.SetVotedFor("")
	}
	n.setState(Follower)
}

// becomeCandidate transitions to candidate state and starts an election
// Must be called with lock held
func (n *Node) becomeCandidate() {
	n.setState(Candidate)

	// Increment current term
	newTerm := n.serverState.GetCurrentTerm() + 1
	n.serverState.SetCurrentTerm(newTerm)

	// Vote for self
	n.serverState.SetVotedFor(n.id)

	// Reset election state
	n.electionState.Reset()
	n.electionState.AddVote(n.id) // Count our own vote

	n.logger.Printf("[INFO] Node %s became candidate for term %d", n.id, newTerm)
}

// becomeLeader transitions to leader state
// Must be called with lock held
func (n *Node) becomeLeader() {
	n.setState(Leader)
	n.logger.Printf("[INFO] Node %s became leader for term %d", n.id, n.serverState.GetCurrentTerm())

	// Append a no-op entry to commit entries from previous terms
	// This is important for the Raft safety property (§5.4.2 and §8)
	go n.appendNoOpEntry()

	// Start sending heartbeats
	go n.startHeartbeatTimer()

	// Send initial heartbeat immediately to establish leadership
	go n.sendHeartbeats()

	// Start log replication
	go n.startReplication()
}

// RPC Handler implementations are in rpc_handlers.go
