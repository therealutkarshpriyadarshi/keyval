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

	// Request deduplication
	sessions map[string]*ClientSession

	// Default timeout for operations
	defaultTimeout time.Duration
}

// ClientSession tracks client request history for deduplication
type ClientSession struct {
	mu              sync.RWMutex
	lastSequence    uint64
	lastResponse    *Response
	lastRequestTime time.Time
}

// NewServer creates a new API server
func NewServer(raftNode *raft.Node) *Server {
	return &Server{
		raftNode:       raftNode,
		logger:         log.Default(),
		sessions:       make(map[string]*ClientSession),
		defaultTimeout: 5 * time.Second,
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
	if cached := s.checkDuplicate(req); cached != nil {
		s.logger.Printf("[DEBUG] Returning cached response for request %s", req.ID)
		return cached
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
	if req.ClientID != "" {
		s.cacheResponse(req, resp)
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

// checkDuplicate checks if this request has already been processed
func (s *Server) checkDuplicate(req *Request) *Response {
	if req.ClientID == "" {
		return nil
	}

	s.mu.RLock()
	session, exists := s.sessions[req.ClientID]
	s.mu.RUnlock()

	if !exists {
		return nil
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// If we've seen this sequence number, return cached response
	if req.Sequence <= session.lastSequence {
		return session.lastResponse
	}

	return nil
}

// cacheResponse caches a response for deduplication
func (s *Server) cacheResponse(req *Request, resp *Response) {
	if req.ClientID == "" {
		return
	}

	s.mu.Lock()
	session, exists := s.sessions[req.ClientID]
	if !exists {
		session = &ClientSession{}
		s.sessions[req.ClientID] = session
	}
	s.mu.Unlock()

	session.mu.Lock()
	defer session.mu.Unlock()

	if req.Sequence > session.lastSequence {
		session.lastSequence = req.Sequence
		session.lastResponse = resp
		session.lastRequestTime = time.Now()
	}
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

// CleanupSessions removes old client sessions
func (s *Server) CleanupSessions(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for clientID, session := range s.sessions {
		session.mu.RLock()
		age := now.Sub(session.lastRequestTime)
		session.mu.RUnlock()

		if age > maxAge {
			delete(s.sessions, clientID)
		}
	}
}
