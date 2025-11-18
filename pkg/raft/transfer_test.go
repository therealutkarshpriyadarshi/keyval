package raft

import (
	"testing"
	"time"
)

// Note: Full integration tests for TransferLeadership, HandleTimeoutNow, etc.
// will be added in Week 9 after Node integration.
// These tests focus on the LeaderTransfer type methods.

func TestLeaderTransfer_Start(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	called := false
	onDone := func(success bool) {
		called = true
	}

	err := lt.Start("node2", onDone)
	if err != nil {
		t.Fatalf("failed to start transfer: %v", err)
	}

	if !lt.IsInProgress() {
		t.Error("transfer should be in progress")
	}

	if lt.GetTarget() != "node2" {
		t.Errorf("expected target node2, got %s", lt.GetTarget())
	}

	if !lt.ShouldStopAcceptingRequests() {
		t.Error("should stop accepting requests during transfer")
	}

	// Try to start another transfer
	err = lt.Start("node3", onDone)
	if err == nil {
		t.Error("expected error when starting transfer while one is in progress")
	}

	// Clean up
	_ = called
}

func TestLeaderTransfer_Complete(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	callbackCalled := false
	callbackSuccess := false

	onDone := func(success bool) {
		callbackCalled = true
		callbackSuccess = success
	}

	lt.Start("node2", onDone)

	// Give callback a moment to potentially run (it shouldn't yet)
	time.Sleep(10 * time.Millisecond)

	if callbackCalled {
		t.Error("callback should not be called before Complete")
	}

	lt.Complete(true)

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	if !callbackCalled {
		t.Error("callback should be called after Complete")
	}

	if !callbackSuccess {
		t.Error("callback should receive success=true")
	}

	if lt.ShouldStopAcceptingRequests() {
		t.Error("should resume accepting requests after complete")
	}

	if lt.IsInProgress() {
		t.Error("transfer should not be in progress after complete")
	}
}

func TestLeaderTransfer_CompleteFailure(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	callbackSuccess := true // Set to true initially
	onDone := func(success bool) {
		callbackSuccess = success
	}

	lt.Start("node2", onDone)
	lt.Complete(false)

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	if callbackSuccess {
		t.Error("callback should receive success=false")
	}
}

func TestLeaderTransfer_IsTimedOut(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 100 * time.Millisecond,
	}

	if lt.IsTimedOut() {
		t.Error("should not be timed out before starting")
	}

	lt.Start("node2", nil)

	if lt.IsTimedOut() {
		t.Error("should not be timed out immediately after starting")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if !lt.IsTimedOut() {
		t.Error("should be timed out after timeout duration")
	}
}

func TestLeaderTransfer_Reset(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	lt.Start("node2", nil)

	if !lt.IsInProgress() {
		t.Error("transfer should be in progress before reset")
	}

	lt.Reset()

	if lt.IsInProgress() {
		t.Error("transfer should not be in progress after reset")
	}

	if lt.GetTarget() != "" {
		t.Error("target should be empty after reset")
	}

	if lt.ShouldStopAcceptingRequests() {
		t.Error("should accept requests after reset")
	}
}

func TestLeaderTransfer_GetElapsedTime(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	elapsed := lt.GetElapsedTime()
	if elapsed != 0 {
		t.Error("elapsed time should be 0 before starting")
	}

	lt.Start("node2", nil)

	time.Sleep(100 * time.Millisecond)

	elapsed = lt.GetElapsedTime()
	if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("elapsed time should be ~100ms, got %v", elapsed)
	}

	lt.Complete(true)

	elapsed = lt.GetElapsedTime()
	if elapsed != 0 {
		t.Error("elapsed time should be 0 after completing")
	}
}

func TestTransferState_Values(t *testing.T) {
	// Verify the constant values are distinct
	states := []TransferState{NoTransfer, TransferInProgress, TransferCompleted, TransferFailed}

	for i, state := range states {
		if int(state) != i {
			t.Errorf("expected state value %d, got %d", i, int(state))
		}
	}
}

func TestLeaderTransfer_MultipleTransfers(t *testing.T) {
	lt := &LeaderTransfer{
		state:   NoTransfer,
		timeout: 5 * time.Second,
	}

	// First transfer
	err := lt.Start("node2", nil)
	if err != nil {
		t.Fatalf("first transfer failed to start: %v", err)
	}

	lt.Complete(true)

	// Second transfer should succeed after first completes
	err = lt.Start("node3", nil)
	if err != nil {
		t.Fatalf("second transfer failed to start: %v", err)
	}

	if lt.GetTarget() != "node3" {
		t.Errorf("expected target node3, got %s", lt.GetTarget())
	}
}

func TestTimeoutNowRequest_Response(t *testing.T) {
	// Test that the request/response types are properly defined
	req := &TimeoutNowRequest{
		Term:     5,
		LeaderId: "leader1",
	}

	if req.Term != 5 {
		t.Errorf("expected term 5, got %d", req.Term)
	}

	if req.LeaderId != "leader1" {
		t.Errorf("expected leader1, got %s", req.LeaderId)
	}

	resp := &TimeoutNowResponse{
		Term: 6,
	}

	if resp.Term != 6 {
		t.Errorf("expected term 6, got %d", resp.Term)
	}
}

func TestNewLeaderTransfer(t *testing.T) {
	// Test with zero timeout (should use default)
	lt := NewLeaderTransfer(0)

	if lt.timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", lt.timeout)
	}

	// Test with explicit timeout
	lt2 := NewLeaderTransfer(5 * time.Second)

	if lt2.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", lt2.timeout)
	}

	// Test initial state
	if lt.state != NoTransfer {
		t.Errorf("expected initial state NoTransfer, got %v", lt.state)
	}

	if lt.stopAcceptingReqs {
		t.Error("should accept requests initially")
	}
}
