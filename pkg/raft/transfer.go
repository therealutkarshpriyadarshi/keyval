package raft

import (
	"fmt"
	"sync"
	"time"
)

// TransferState represents the state of a leadership transfer
type TransferState int

const (
	// NoTransfer indicates no transfer is in progress
	NoTransfer TransferState = iota
	// TransferInProgress indicates a transfer is currently happening
	TransferInProgress
	// TransferCompleted indicates the transfer completed successfully
	TransferCompleted
	// TransferFailed indicates the transfer failed
	TransferFailed
)

// LeaderTransfer manages the leadership transfer process
type LeaderTransfer struct {
	mu               sync.RWMutex
	state            TransferState
	targetID         string
	startTime        time.Time
	timeout          time.Duration
	stopAcceptingReqs bool
	onTransferDone   func(success bool)
}

// NewLeaderTransfer creates a new leader transfer manager
func NewLeaderTransfer(timeout time.Duration) *LeaderTransfer {
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}

	return &LeaderTransfer{
		state:            NoTransfer,
		timeout:          timeout,
		stopAcceptingReqs: false,
	}
}

// Start initiates a leadership transfer to the target node
func (lt *LeaderTransfer) Start(targetID string, onDone func(success bool)) error {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if lt.state == TransferInProgress {
		return fmt.Errorf("leadership transfer already in progress to %s", lt.targetID)
	}

	lt.state = TransferInProgress
	lt.targetID = targetID
	lt.startTime = time.Now()
	lt.stopAcceptingReqs = true
	lt.onTransferDone = onDone

	return nil
}

// IsInProgress returns whether a transfer is currently in progress
func (lt *LeaderTransfer) IsInProgress() bool {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.state == TransferInProgress
}

// GetTarget returns the target node ID for the transfer
func (lt *LeaderTransfer) GetTarget() string {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.targetID
}

// ShouldStopAcceptingRequests returns whether the leader should stop accepting new client requests
func (lt *LeaderTransfer) ShouldStopAcceptingRequests() bool {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.stopAcceptingReqs
}

// IsTimedOut returns whether the transfer has timed out
func (lt *LeaderTransfer) IsTimedOut() bool {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if lt.state != TransferInProgress {
		return false
	}

	return time.Since(lt.startTime) > lt.timeout
}

// Complete marks the transfer as completed
func (lt *LeaderTransfer) Complete(success bool) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if success {
		lt.state = TransferCompleted
	} else {
		lt.state = TransferFailed
	}

	lt.stopAcceptingReqs = false

	if lt.onTransferDone != nil {
		go lt.onTransferDone(success)
	}
}

// Reset resets the transfer state
func (lt *LeaderTransfer) Reset() {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	lt.state = NoTransfer
	lt.targetID = ""
	lt.stopAcceptingReqs = false
	lt.onTransferDone = nil
}

// GetElapsedTime returns how long the transfer has been running
func (lt *LeaderTransfer) GetElapsedTime() time.Duration {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if lt.state != TransferInProgress {
		return 0
	}

	return time.Since(lt.startTime)
}

// TransferLeadership initiates a leadership transfer on the node
func (n *Node) TransferLeadership(targetID string) error {
	n.mu.Lock()

	// Verify we are the leader
	if n.state != Leader {
		n.mu.Unlock()
		return fmt.Errorf("only leader can transfer leadership")
	}

	// Verify target exists in configuration
	config := n.getConfiguration()
	if !config.IsVoter(targetID) {
		n.mu.Unlock()
		return fmt.Errorf("target %s is not a voting member", targetID)
	}

	// Check if transfer already in progress
	if n.leaderTransfer != nil && n.leaderTransfer.IsInProgress() {
		n.mu.Unlock()
		return fmt.Errorf("leadership transfer already in progress")
	}

	// Initialize transfer if needed
	if n.leaderTransfer == nil {
		n.leaderTransfer = NewLeaderTransfer(10 * time.Second)
	}

	currentTerm := n.serverState.currentTerm
	n.mu.Unlock()

	// Start the transfer
	err := n.leaderTransfer.Start(targetID, func(success bool) {
		if success {
			n.logger.Info("leadership transfer completed successfully", "target", targetID)
		} else {
			n.logger.Warn("leadership transfer failed", "target", targetID)
			// Resume accepting requests on failure
			n.leaderTransfer.Reset()
		}
	})

	if err != nil {
		return err
	}

	n.logger.Info("initiating leadership transfer", "target", targetID, "term", currentTerm)

	// Start the transfer process in a goroutine
	go n.executeLeadershipTransfer(targetID, currentTerm)

	return nil
}

// executeLeadershipTransfer executes the leadership transfer process
func (n *Node) executeLeadershipTransfer(targetID string, term uint64) {
	// Step 1: Ensure target is caught up
	n.logger.Info("ensuring target is caught up", "target", targetID)

	timeout := time.After(n.leaderTransfer.timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			n.logger.Warn("leadership transfer timed out", "target", targetID)
			n.leaderTransfer.Complete(false)
			return

		case <-ticker.C:
			n.mu.RLock()

			// Check if we're still leader
			if n.state != Leader || n.serverState.currentTerm != term {
				n.mu.RUnlock()
				n.logger.Warn("no longer leader, aborting transfer", "target", targetID)
				n.leaderTransfer.Complete(false)
				return
			}

			// Check if target is caught up
			leaderState := n.leaderState
			lastLogIndex := n.serverState.log.LastIndex()
			n.mu.RUnlock()

			if leaderState == nil {
				n.logger.Warn("leader state is nil, aborting transfer")
				n.leaderTransfer.Complete(false)
				return
			}

			leaderState.mu.RLock()
			matchIndex, exists := leaderState.matchIndex[targetID]
			leaderState.mu.RUnlock()

			if !exists {
				n.logger.Warn("target not found in matchIndex", "target", targetID)
				continue
			}

			// Check if target is caught up (within a small threshold)
			if lastLogIndex-matchIndex <= 10 {
				n.logger.Info("target is caught up, proceeding with transfer",
					"target", targetID,
					"matchIndex", matchIndex,
					"lastLogIndex", lastLogIndex)
				goto CaughtUp
			}

			n.logger.Debug("waiting for target to catch up",
				"target", targetID,
				"matchIndex", matchIndex,
				"lastLogIndex", lastLogIndex,
				"gap", lastLogIndex-matchIndex)
		}
	}

CaughtUp:
	// Step 2: Stop accepting new client requests
	n.logger.Info("stopping acceptance of new client requests", "target", targetID)

	// Wait a bit to ensure in-flight requests are processed
	time.Sleep(100 * time.Millisecond)

	// Step 3: Send TimeoutNow RPC to target
	n.logger.Info("sending TimeoutNow RPC to target", "target", targetID)

	n.mu.RLock()
	currentTerm := n.serverState.currentTerm
	n.mu.RUnlock()

	success := n.sendTimeoutNowRPC(targetID, currentTerm)

	if success {
		// Step 4: Step down to follower
		n.logger.Info("stepping down after successful transfer", "target", targetID)
		n.mu.Lock()
		n.becomeFollower(currentTerm)
		n.mu.Unlock()

		n.leaderTransfer.Complete(true)
	} else {
		n.logger.Warn("failed to send TimeoutNow RPC", "target", targetID)
		n.leaderTransfer.Complete(false)
	}
}

// sendTimeoutNowRPC sends a TimeoutNow RPC to the target node
func (n *Node) sendTimeoutNowRPC(targetID string, term uint64) bool {
	// Get target address
	n.mu.RLock()
	config := n.getConfiguration()
	n.mu.RUnlock()

	var targetAddress string
	for id, member := range config.Members {
		if id == targetID {
			targetAddress = member.Address
			break
		}
	}

	if targetAddress == "" {
		n.logger.Warn("target address not found", "target", targetID)
		return false
	}

	// Send TimeoutNow RPC
	request := &TimeoutNowRequest{
		Term:     term,
		LeaderId: n.id,
	}

	response, err := n.rpcClient.TimeoutNow(targetAddress, request)
	if err != nil {
		n.logger.Warn("TimeoutNow RPC failed", "target", targetID, "error", err)
		return false
	}

	if response.Term > term {
		// Target has moved to a higher term
		n.mu.Lock()
		n.becomeFollower(response.Term)
		n.mu.Unlock()
		return false
	}

	return true
}

// HandleTimeoutNow handles a TimeoutNow RPC from the current leader
func (n *Node) HandleTimeoutNow(request *TimeoutNowRequest) (*TimeoutNowResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.currentTerm

	// If request term is less than current term, reject
	if request.Term < currentTerm {
		return &TimeoutNowResponse{
			Term: currentTerm,
		}, nil
	}

	// If request term is greater, update term
	if request.Term > currentTerm {
		n.becomeFollower(request.Term)
		currentTerm = request.Term
	}

	// Only followers and candidates can receive TimeoutNow
	if n.state != Follower && n.state != Candidate {
		return &TimeoutNowResponse{
			Term: currentTerm,
		}, nil
	}

	n.logger.Info("received TimeoutNow RPC, starting immediate election",
		"from", request.LeaderId,
		"term", request.Term)

	// Immediately start an election
	go n.startElection()

	return &TimeoutNowResponse{
		Term: currentTerm,
	}, nil
}

// TimeoutNowRequest is sent by leader to transfer leadership
type TimeoutNowRequest struct {
	Term     uint64 // Leader's current term
	LeaderId string // Leader's ID
}

// TimeoutNowResponse is the response to TimeoutNow RPC
type TimeoutNowResponse struct {
	Term uint64 // Receiver's current term
}

// AbortLeadershipTransfer aborts an ongoing leadership transfer
func (n *Node) AbortLeadershipTransfer() {
	if n.leaderTransfer != nil {
		n.leaderTransfer.Complete(false)
	}
}

// IsLeadershipTransferInProgress returns whether a leadership transfer is in progress
func (n *Node) IsLeadershipTransferInProgress() bool {
	if n.leaderTransfer == nil {
		return false
	}
	return n.leaderTransfer.IsInProgress()
}
