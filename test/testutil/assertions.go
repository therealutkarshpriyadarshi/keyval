package testutil

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
)

// AssertLeaderElected asserts that a leader is elected within timeout
func AssertLeaderElected(t *testing.T, cluster *TestCluster, timeout time.Duration) *raft.Node {
	t.Helper()

	leader := cluster.WaitForLeader(timeout)
	if leader == nil {
		t.Fatal("Expected leader to be elected")
	}

	return leader
}

// AssertSingleLeader asserts that exactly one leader exists in the cluster
func AssertSingleLeader(t *testing.T, cluster *TestCluster) *raft.Node {
	t.Helper()

	count := cluster.CountLeaders()
	if count != 1 {
		t.Fatalf("Expected 1 leader, got %d", count)
	}

	return cluster.GetLeader()
}

// AssertNoLeader asserts that no leader exists in the cluster
func AssertNoLeader(t *testing.T, cluster *TestCluster) {
	t.Helper()

	count := cluster.CountLeaders()
	if count != 0 {
		t.Fatalf("Expected no leader, got %d", count)
	}
}

// AssertTerm asserts that a node is in the expected term
func AssertTerm(t *testing.T, node *raft.Node, expectedTerm uint64) {
	t.Helper()

	actualTerm := node.GetCurrentTerm()
	if actualTerm != expectedTerm {
		t.Fatalf("Expected term %d, got %d", expectedTerm, actualTerm)
	}
}

// AssertCommitIndex asserts that a node has the expected commit index
func AssertCommitIndex(t *testing.T, node *raft.Node, expectedCommit uint64) {
	t.Helper()

	actualCommit := node.GetCommitIndex()
	if actualCommit != expectedCommit {
		t.Fatalf("Expected commit index %d, got %d", expectedCommit, actualCommit)
	}
}

// AssertConvergence asserts that all nodes converge to same state within timeout
func AssertConvergence(t *testing.T, cluster *TestCluster, timeout time.Duration) {
	t.Helper()

	cluster.WaitForConvergence(timeout)
}

// AssertEventually asserts that a condition becomes true within timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Condition not met within %v: %s", timeout, message)
}

// AssertNever asserts that a condition never becomes true within duration
func AssertNever(t *testing.T, condition func() bool, duration time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if condition() {
			t.Fatalf("Condition unexpectedly met: %s", message)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// AssertEqual asserts that two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, message string) {
	t.Helper()

	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", message, expected, actual)
	}
}

// AssertNotEqual asserts that two values are not equal
func AssertNotEqual(t *testing.T, unexpected, actual interface{}, message string) {
	t.Helper()

	if unexpected == actual {
		t.Fatalf("%s: expected not %v, but got it", message, unexpected)
	}
}

// AssertTrue asserts that a condition is true
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()

	if !condition {
		t.Fatalf("Expected true: %s", message)
	}
}

// AssertFalse asserts that a condition is false
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()

	if condition {
		t.Fatalf("Expected false: %s", message)
	}
}

// AssertNil asserts that a value is nil
func AssertNil(t *testing.T, value interface{}, message string) {
	t.Helper()

	if value != nil {
		t.Fatalf("Expected nil: %s, got %v", message, value)
	}
}

// AssertNotNil asserts that a value is not nil
func AssertNotNil(t *testing.T, value interface{}, message string) {
	t.Helper()

	if value == nil {
		t.Fatalf("Expected non-nil: %s", message)
	}
}

// AssertError asserts that an error occurred
func AssertError(t *testing.T, err error, message string) {
	t.Helper()

	if err == nil {
		t.Fatalf("Expected error: %s", message)
	}
}

// AssertNoError asserts that no error occurred
func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()

	if err != nil {
		t.Fatalf("Unexpected error: %s: %v", message, err)
	}
}
