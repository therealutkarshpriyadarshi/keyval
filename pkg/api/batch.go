package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// BatchRequest represents a batch of operations
type BatchRequest struct {
	// Unique request ID
	ID string

	// Client ID for idempotency
	ClientID string

	// Sequence number
	Sequence uint64

	// Operations in the batch
	Operations []Operation

	// Timeout for the batch
	Timeout time.Duration
}

// Operation represents a single operation in a batch
type Operation struct {
	// Operation type
	Type OperationType

	// Key for the operation
	Key string

	// Value for Put operations
	Value []byte
}

// BatchResponse represents the response to a batch request
type BatchResponse struct {
	// Success indicates if the batch succeeded
	Success bool

	// Results for each operation
	Results []OperationResult

	// Error message if batch failed
	Error string

	// LeaderID if redirection needed
	LeaderID string

	// LeaderAddress for client redirection
	LeaderAddress string
}

// OperationResult represents the result of a single operation
type OperationResult struct {
	// Success indicates if this operation succeeded
	Success bool

	// Value returned for Get operations
	Value []byte

	// Error message if operation failed
	Error string
}

// BatchCommand represents a batch of commands for the state machine
type BatchCommand struct {
	Commands []statemachine.Command
}

// HandleBatch processes a batch request
func (s *Server) HandleBatch(req *BatchRequest) *BatchResponse {
	// Validate batch request
	if err := s.validateBatchRequest(req); err != nil {
		return &BatchResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Check for duplicate request
	if req.ClientID != "" && req.Sequence > 0 {
		if cached, isDup := s.sessionManager.CheckDuplicate(req.ClientID, req.Sequence); isDup {
			s.logger.Printf("[DEBUG] Returning cached batch response for request %s", req.ID)
			// Convert cached Response to BatchResponse
			return convertToBatchResponse(cached)
		}
	}

	// Only leader can process batch writes
	// Check if any operations are writes
	hasWrites := false
	for _, op := range req.Operations {
		if op.Type == OperationPut || op.Type == OperationDelete {
			hasWrites = true
			break
		}
	}

	if hasWrites && !s.raftNode.IsLeader() {
		return s.redirectBatchToLeader()
	}

	// Separate reads and writes
	var writes []Operation
	var reads []Operation

	for _, op := range req.Operations {
		if op.Type == OperationGet {
			reads = append(reads, op)
		} else {
			writes = append(writes, op)
		}
	}

	resp := &BatchResponse{
		Success: true,
		Results: make([]OperationResult, 0, len(req.Operations)),
	}

	// Process writes as a single batch
	if len(writes) > 0 {
		writeResults, err := s.processBatchWrites(writes, req.Timeout)
		if err != nil {
			return &BatchResponse{
				Success: false,
				Error:   err.Error(),
			}
		}
		resp.Results = append(resp.Results, writeResults...)
	}

	// Process reads individually (they're fast)
	if len(reads) > 0 {
		readResults := s.processBatchReads(reads)
		resp.Results = append(resp.Results, readResults...)
	}

	// Cache response
	if req.ClientID != "" && req.Sequence > 0 {
		// Convert to regular Response for caching
		cachedResp := &Response{
			Success: resp.Success,
			Error:   resp.Error,
		}
		s.sessionManager.CacheResponse(req.ClientID, req.Sequence, cachedResp)
	}

	return resp
}

// processBatchWrites processes a batch of write operations
func (s *Server) processBatchWrites(writes []Operation, timeout time.Duration) ([]OperationResult, error) {
	if !s.raftNode.IsLeader() {
		return nil, fmt.Errorf(ErrNotLeader)
	}

	// Create batch command
	batchCmd := BatchCommand{
		Commands: make([]statemachine.Command, len(writes)),
	}

	for i, op := range writes {
		cmd := statemachine.Command{
			Key:   op.Key,
			Value: op.Value,
		}

		switch op.Type {
		case OperationPut:
			cmd.Type = statemachine.CommandPut
		case OperationDelete:
			cmd.Type = statemachine.CommandDelete
		default:
			return nil, fmt.Errorf("invalid operation type in batch: %v", op.Type)
		}

		batchCmd.Commands[i] = cmd
	}

	// Encode batch command
	data, err := json.Marshal(batchCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to encode batch command: %v", err)
	}

	// Append to Raft log
	index, isLeader := s.raftNode.AppendLogEntry(data, raft.EntryBatch)
	if !isLeader {
		return nil, fmt.Errorf(ErrNotLeader)
	}

	// Wait for commit
	if timeout == 0 {
		timeout = s.defaultTimeout
	}

	committed := s.waitForCommit(index, timeout)
	if !committed {
		return nil, fmt.Errorf(ErrTimeout)
	}

	// Create success results for each write
	results := make([]OperationResult, len(writes))
	for i := range writes {
		results[i] = OperationResult{
			Success: true,
		}
	}

	return results, nil
}

// processBatchReads processes a batch of read operations
func (s *Server) processBatchReads(reads []Operation) []OperationResult {
	results := make([]OperationResult, len(reads))

	for i, op := range reads {
		value, exists := s.getFromStateMachine(op.Key)
		if !exists {
			results[i] = OperationResult{
				Success: false,
				Error:   ErrKeyNotFound,
			}
		} else {
			results[i] = OperationResult{
				Success: true,
				Value:   value,
			}
		}
	}

	return results
}

// validateBatchRequest validates a batch request
func (s *Server) validateBatchRequest(req *BatchRequest) error {
	if req == nil {
		return fmt.Errorf("batch request is nil")
	}

	if len(req.Operations) == 0 {
		return fmt.Errorf("batch must contain at least one operation")
	}

	if len(req.Operations) > 1000 {
		return fmt.Errorf("batch size exceeds maximum of 1000 operations")
	}

	// Validate each operation
	for i, op := range req.Operations {
		if op.Key == "" {
			return fmt.Errorf("operation %d: key cannot be empty", i)
		}

		if op.Type == OperationPut && op.Value == nil {
			return fmt.Errorf("operation %d: value cannot be nil for Put operation", i)
		}
	}

	return nil
}

// redirectBatchToLeader returns a response redirecting to the leader
func (s *Server) redirectBatchToLeader() *BatchResponse {
	leaderID := s.raftNode.GetLeaderID()

	return &BatchResponse{
		Success:  false,
		Error:    ErrNotLeader,
		LeaderID: leaderID,
	}
}

// convertToBatchResponse converts a regular Response to BatchResponse
func convertToBatchResponse(resp *Response) *BatchResponse {
	return &BatchResponse{
		Success:       resp.Success,
		Error:         resp.Error,
		LeaderID:      resp.LeaderID,
		LeaderAddress: resp.LeaderAddress,
	}
}
