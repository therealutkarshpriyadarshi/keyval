package api

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// Server handles client API requests
type Server struct {
	mu       sync.RWMutex
	raftNode *raft.Node
	logger   *log.Logger

	// Session manager for request deduplication
	sessionManager *SessionManager

	// Default timeout for operations
	defaultTimeout time.Duration
}

// NewServer creates a new API server
func NewServer(raftNode *raft.Node) *Server {
	return &Server{
		raftNode:       raftNode,
		logger:         log.Default(),
		sessionManager: NewSessionManager(10 * time.Minute), // 10 minute session timeout
		defaultTimeout: 5 * time.Second,
	}
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	if s.sessionManager != nil {
		s.sessionManager.Stop()
	}
}

// HandleRequest processes a client request
func (s *Server) HandleRequest(req *Request) *Response {
	// Validate request
	if err := s.validateRequest(req); err != nil {
		return &Response{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Check for duplicate request (idempotency)
	if req.ClientID != "" && req.Sequence > 0 {
		if cached, isDup := s.sessionManager.CheckDuplicate(req.ClientID, req.Sequence); isDup {
			s.logger.Printf("[DEBUG] Returning cached response for request %s (client=%s, seq=%d)",
				req.ID, req.ClientID, req.Sequence)
			return cached
		}
	}

	// Route based on operation type
	var resp *Response
	switch req.Type {
	case OperationGet:
		resp = s.handleGet(req)
	case OperationPut:
		resp = s.handlePut(req)
	case OperationDelete:
		resp = s.handleDelete(req)
	default:
		resp = &Response{
			Success: false,
			Error:   fmt.Sprintf("unknown operation type: %v", req.Type),
		}
	}

	// Cache response for deduplication
	if req.ClientID != "" && req.Sequence > 0 {
		s.sessionManager.CacheResponse(req.ClientID, req.Sequence, resp)
	}

	return resp
}

// handleGet processes a Get request
func (s *Server) handleGet(req *Request) *Response {
	// For linearizable reads, we need to ensure we're still the leader
	// and that we have up-to-date information
	if !s.raftNode.IsLeader() {
		return s.redirectToLeader()
	}

	// Wait for a heartbeat round to ensure we're still leader
	// This implements the ReadIndex optimization
	time.Sleep(100 * time.Millisecond) // Simple approach for now

	if !s.raftNode.IsLeader() {
		return s.redirectToLeader()
	}

	// Read from state machine
	value, exists := s.getFromStateMachine(req.Key)
	if !exists {
		return &Response{
			Success: false,
			Error:   ErrKeyNotFound,
		}
	}

	return &Response{
		Success: true,
		Value:   value,
	}
}

// handlePut processes a Put request
func (s *Server) handlePut(req *Request) *Response {
	// Only leader can process writes
	if !s.raftNode.IsLeader() {
		return s.redirectToLeader()
	}

	// Create command for state machine
	cmd := statemachine.Command{
		Type:  statemachine.CommandPut,
		Key:   req.Key,
		Value: req.Value,
	}

	// Encode command
	data, err := json.Marshal(cmd)
	if err != nil {
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("failed to encode command: %v", err),
		}
	}

	// Append to Raft log
	index, isLeader := s.raftNode.AppendLogEntry(data, raft.EntryNormal)
	if !isLeader {
		return s.redirectToLeader()
	}

	// Wait for commit
	timeout := req.Timeout
	if timeout == 0 {
		timeout = s.defaultTimeout
	}

	committed := s.waitForCommit(index, timeout)
	if !committed {
		return &Response{
			Success: false,
			Error:   ErrTimeout,
		}
	}

	return &Response{
		Success: true,
		Value:   req.Value,
	}
}

// handleDelete processes a Delete request
func (s *Server) handleDelete(req *Request) *Response {
	// Only leader can process writes
	if !s.raftNode.IsLeader() {
		return s.redirectToLeader()
	}

	// Create command for state machine
	cmd := statemachine.Command{
		Type: statemachine.CommandDelete,
		Key:  req.Key,
	}

	// Encode command
	data, err := json.Marshal(cmd)
	if err != nil {
		return &Response{
			Success: false,
			Error:   fmt.Sprintf("failed to encode command: %v", err),
		}
	}

	// Append to Raft log
	index, isLeader := s.raftNode.AppendLogEntry(data, raft.EntryNormal)
	if !isLeader {
		return s.redirectToLeader()
	}

	// Wait for commit
	timeout := req.Timeout
	if timeout == 0 {
		timeout = s.defaultTimeout
	}

	committed := s.waitForCommit(index, timeout)
	if !committed {
		return &Response{
			Success: false,
			Error:   ErrTimeout,
		}
	}

	return &Response{
		Success: true,
	}
}

// validateRequest validates a client request
func (s *Server) validateRequest(req *Request) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if req.Key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	if req.Type == OperationPut && req.Value == nil {
		return fmt.Errorf("value cannot be nil for Put operation")
	}

	return nil
}


// redirectToLeader returns a response indicating the client should redirect to the leader
func (s *Server) redirectToLeader() *Response {
	leaderID := s.raftNode.GetLeaderID()

	return &Response{
		Success:   false,
		Error:     ErrNotLeader,
		LeaderID:  leaderID,
		// LeaderAddress would be populated if we track peer addresses
	}
}

// waitForCommit waits for a log index to be committed
func (s *Server) waitForCommit(index uint64, timeout time.Duration) bool {
	deadline := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return false
		case <-ticker.C:
			if s.raftNode.GetCommitIndex() >= index {
				// Also wait for it to be applied
				if s.raftNode.GetLastApplied() >= index {
					return true
				}
			}

			// Check if we're still the leader
			if !s.raftNode.IsLeader() {
				return false
			}
		}
	}
}

// getFromStateMachine retrieves a value from the state machine
func (s *Server) getFromStateMachine(key string) ([]byte, bool) {
	// Use the linearizable read method from Raft node
	value, exists, err := s.raftNode.Get(key)
	if err != nil {
		s.logger.Printf("[ERROR] Failed to get key %s: %v", key, err)
		return nil, false
	}
	return value, exists
}

// GetSessionCount returns the number of active sessions
func (s *Server) GetSessionCount() int {
	return s.sessionManager.SessionCount()
}
