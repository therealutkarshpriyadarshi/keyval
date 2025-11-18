// +build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
)

// createTestCluster creates a cluster of Raft nodes for testing
func createTestCluster(t *testing.T, size int) []*raft.Node {
	if size < 1 {
		t.Fatal("Cluster size must be at least 1")
	}

	nodes := make([]*raft.Node, size)
	basePort := 10000

	// Create peer configurations for all nodes
	allPeers := make([]raft.Peer, size)
	for i := 0; i < size; i++ {
		allPeers[i] = raft.Peer{
			ID:      fmt.Sprintf("node%d", i+1),
			Address: fmt.Sprintf("localhost:%d", basePort+i),
		}
	}

	// Create each node
	for i := 0; i < size; i++ {
		// Peers for this node (all except self)
		peers := make([]raft.Peer, 0, size-1)
		for j := 0; j < size; j++ {
			if i != j {
				peers = append(peers, allPeers[j])
			}
		}

		node, err := raft.NewNode(
			allPeers[i].ID,
			allPeers[i].Address,
			peers,
			raft.WithConfig(raft.FastConfig()),
		)
		if err != nil {
			t.Fatalf("Failed to create node %d: %v", i+1, err)
		}

		nodes[i] = node
	}

	return nodes
}

// startCluster starts all nodes in the cluster
func startCluster(t *testing.T, nodes []*raft.Node) {
	for _, node := range nodes {
		if err := node.Start(); err != nil {
			t.Fatalf("Failed to start node %s: %v", node.GetID(), err)
		}
	}

	// Give nodes time to connect
	time.Sleep(200 * time.Millisecond)
}

// stopCluster stops all nodes in the cluster
func stopCluster(nodes []*raft.Node) {
	for _, node := range nodes {
		node.Stop()
	}
}

// waitForLeader waits for a leader to be elected and returns the leader
func waitForLeader(t *testing.T, nodes []*raft.Node, timeout time.Duration) *raft.Node {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		for _, node := range nodes {
			if node.IsLeader() {
				return node
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("No leader elected within timeout")
	return nil
}

// countLeaders counts the number of leaders in the cluster
func countLeaders(nodes []*raft.Node) int {
	count := 0
	for _, node := range nodes {
		if node.IsLeader() {
			count++
		}
	}
	return count
}

// TestSingleNodeElection tests that a single node elects itself as leader
func TestSingleNodeElection(t *testing.T) {
	nodes := createTestCluster(t, 1)
	defer stopCluster(nodes)

	startCluster(t, nodes)

	// Wait for the node to become leader
	leader := waitForLeader(t, nodes, 2*time.Second)

	if leader.GetID() != "node1" {
		t.Errorf("Expected node1 to be leader, got %s", leader.GetID())
	}

	if leader.GetCurrentTerm() < 1 {
		t.Errorf("Leader term should be at least 1, got %d", leader.GetCurrentTerm())
	}
}

// TestThreeNodeElection tests leader election in a 3-node cluster
func TestThreeNodeElection(t *testing.T) {
	nodes := createTestCluster(t, 3)
	defer stopCluster(nodes)

	startCluster(t, nodes)

	// Wait for a leader to be elected
	leader := waitForLeader(t, nodes, 3*time.Second)

	// Should have exactly one leader
	leaderCount := countLeaders(nodes)
	if leaderCount != 1 {
		t.Errorf("Expected 1 leader, got %d", leaderCount)
	}

	// All nodes should be in the same term
	term := leader.GetCurrentTerm()
	for _, node := range nodes {
		if node.GetCurrentTerm() != term {
			t.Errorf("Node %s has term %d, expected %d",
				node.GetID(), node.GetCurrentTerm(), term)
		}
	}

	t.Logf("Leader elected: %s in term %d", leader.GetID(), term)
}

// TestFiveNodeElection tests leader election in a 5-node cluster
func TestFiveNodeElection(t *testing.T) {
	nodes := createTestCluster(t, 5)
	defer stopCluster(nodes)

	startCluster(t, nodes)

	// Wait for a leader to be elected
	leader := waitForLeader(t, nodes, 5*time.Second)

	// Should have exactly one leader
	leaderCount := countLeaders(nodes)
	if leaderCount != 1 {
		t.Errorf("Expected 1 leader, got %d", leaderCount)
	}

	// Verify all nodes agree on the term
	term := leader.GetCurrentTerm()
	for _, node := range nodes {
		nodeTerm := node.GetCurrentTerm()
		if nodeTerm < term {
			t.Errorf("Node %s has term %d, which is less than leader term %d",
				node.GetID(), nodeTerm, term)
		}
	}

	t.Logf("Leader elected: %s in term %d", leader.GetID(), term)
}

// TestLeaderStability tests that the leader remains stable without failures
func TestLeaderStability(t *testing.T) {
	nodes := createTestCluster(t, 3)
	defer stopCluster(nodes)

	startCluster(t, nodes)

	// Wait for initial leader
	leader := waitForLeader(t, nodes, 3*time.Second)
	initialLeaderID := leader.GetID()
	initialTerm := leader.GetCurrentTerm()

	t.Logf("Initial leader: %s in term %d", initialLeaderID, initialTerm)

	// Wait for a while and ensure leadership doesn't change
	time.Sleep(2 * time.Second)

	// Should still have the same leader
	leaderCount := countLeaders(nodes)
	if leaderCount != 1 {
		t.Errorf("Expected 1 leader after stability period, got %d", leaderCount)
	}

	// Find current leader
	var currentLeader *raft.Node
	for _, node := range nodes {
		if node.IsLeader() {
			currentLeader = node
			break
		}
	}

	if currentLeader == nil {
		t.Fatal("No leader found after stability period")
	}

	if currentLeader.GetID() != initialLeaderID {
		t.Errorf("Leader changed from %s to %s", initialLeaderID, currentLeader.GetID())
	}

	if currentLeader.GetCurrentTerm() != initialTerm {
		t.Errorf("Term changed from %d to %d", initialTerm, currentLeader.GetCurrentTerm())
	}
}
