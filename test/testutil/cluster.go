package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// TestCluster represents a test cluster of Raft nodes
type TestCluster struct {
	t          *testing.T
	nodes      []*raft.Node
	dataDirs   []string
	basePort   int
	size       int
	shutdownFn func()
}

// ClusterConfig holds configuration for test cluster
type ClusterConfig struct {
	Size                int
	BasePort            int
	ElectionTimeoutMin  time.Duration
	ElectionTimeoutMax  time.Duration
	HeartbeatInterval   time.Duration
	SnapshotInterval    int
	SnapshotThreshold   int
	MaxEntriesPerAppend int
}

// DefaultClusterConfig returns a default test cluster configuration
func DefaultClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		Size:                3,
		BasePort:            10000,
		ElectionTimeoutMin:  150 * time.Millisecond,
		ElectionTimeoutMax:  300 * time.Millisecond,
		HeartbeatInterval:   50 * time.Millisecond,
		SnapshotInterval:    100,
		SnapshotThreshold:   1000,
		MaxEntriesPerAppend: 100,
	}
}

// FastClusterConfig returns a fast configuration for quick tests
func FastClusterConfig() *ClusterConfig {
	return &ClusterConfig{
		Size:                3,
		BasePort:            10000,
		ElectionTimeoutMin:  50 * time.Millisecond,
		ElectionTimeoutMax:  100 * time.Millisecond,
		HeartbeatInterval:   20 * time.Millisecond,
		SnapshotInterval:    50,
		SnapshotThreshold:   500,
		MaxEntriesPerAppend: 50,
	}
}

// NewTestCluster creates a new test cluster with the given configuration
func NewTestCluster(t *testing.T, config *ClusterConfig) *TestCluster {
	t.Helper()

	if config == nil {
		config = DefaultClusterConfig()
	}

	cluster := &TestCluster{
		t:        t,
		nodes:    make([]*raft.Node, config.Size),
		dataDirs: make([]string, config.Size),
		basePort: config.BasePort,
		size:     config.Size,
	}

	// Create temporary data directories for each node
	for i := 0; i < config.Size; i++ {
		dataDir := t.TempDir()
		cluster.dataDirs[i] = dataDir
	}

	// Build peer list
	peers := make([]string, config.Size)
	for i := 0; i < config.Size; i++ {
		peers[i] = fmt.Sprintf("node%d", i+1)
	}

	// Create nodes
	for i := 0; i < config.Size; i++ {
		nodeID := fmt.Sprintf("node%d", i+1)

		// Create peers list (excluding self)
		nodePeers := make([]string, 0, config.Size-1)
		for j := 0; j < config.Size; j++ {
			if i != j {
				nodePeers = append(nodePeers, fmt.Sprintf("node%d", j+1))
			}
		}

		// Create Raft config
		raftConfig := raft.DefaultConfig()
		raftConfig.ElectionTimeoutMin = config.ElectionTimeoutMin
		raftConfig.ElectionTimeoutMax = config.ElectionTimeoutMax
		raftConfig.HeartbeatInterval = config.HeartbeatInterval
		raftConfig.SnapshotInterval = config.SnapshotInterval
		raftConfig.SnapshotThreshold = config.SnapshotThreshold
		raftConfig.MaxEntriesPerAppend = config.MaxEntriesPerAppend

		// Create state machine
		sm := statemachine.NewKVStateMachine()

		// Create node
		node := raft.NewNode(nodeID, nodePeers, raftConfig, sm)
		cluster.nodes[i] = node
	}

	return cluster
}

// Start starts all nodes in the cluster
func (tc *TestCluster) Start() {
	tc.t.Helper()

	for i, node := range tc.nodes {
		if err := node.Start(); err != nil {
			tc.t.Fatalf("Failed to start node %d: %v", i, err)
		}
	}

	// Give nodes time to connect and stabilize
	time.Sleep(100 * time.Millisecond)
}

// Stop stops all nodes in the cluster
func (tc *TestCluster) Stop() {
	tc.t.Helper()

	for _, node := range tc.nodes {
		node.Stop()
	}

	// Clean up data directories
	for _, dir := range tc.dataDirs {
		os.RemoveAll(dir)
	}
}

// WaitForLeader waits for a leader to be elected within the timeout
func (tc *TestCluster) WaitForLeader(timeout time.Duration) *raft.Node {
	tc.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, node := range tc.nodes {
			if node.IsLeader() {
				return node
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	tc.t.Fatalf("No leader elected within %v", timeout)
	return nil
}

// GetLeader returns the current leader, or nil if none
func (tc *TestCluster) GetLeader() *raft.Node {
	for _, node := range tc.nodes {
		if node.IsLeader() {
			return node
		}
	}
	return nil
}

// CountLeaders returns the number of nodes that think they are leader
func (tc *TestCluster) CountLeaders() int {
	count := 0
	for _, node := range tc.nodes {
		if node.IsLeader() {
			count++
		}
	}
	return count
}

// GetNode returns the node at the given index
func (tc *TestCluster) GetNode(index int) *raft.Node {
	if index < 0 || index >= len(tc.nodes) {
		tc.t.Fatalf("Invalid node index: %d", index)
	}
	return tc.nodes[index]
}

// GetNodes returns all nodes in the cluster
func (tc *TestCluster) GetNodes() []*raft.Node {
	return tc.nodes
}

// Size returns the cluster size
func (tc *TestCluster) Size() int {
	return tc.size
}

// StopNode stops a specific node
func (tc *TestCluster) StopNode(index int) {
	tc.t.Helper()

	if index < 0 || index >= len(tc.nodes) {
		tc.t.Fatalf("Invalid node index: %d", index)
	}

	tc.nodes[index].Stop()
}

// StartNode restarts a specific node
func (tc *TestCluster) StartNode(index int) {
	tc.t.Helper()

	if index < 0 || index >= len(tc.nodes) {
		tc.t.Fatalf("Invalid node index: %d", index)
	}

	if err := tc.nodes[index].Start(); err != nil {
		tc.t.Fatalf("Failed to start node %d: %v", index, err)
	}
}

// WaitForConvergence waits for all nodes to agree on the same term and commit index
func (tc *TestCluster) WaitForConvergence(timeout time.Duration) {
	tc.t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		leader := tc.GetLeader()
		if leader == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		leaderTerm := leader.GetCurrentTerm()
		leaderCommit := leader.GetCommitIndex()

		allConverged := true
		for _, node := range tc.nodes {
			if node.GetCurrentTerm() != leaderTerm {
				allConverged = false
				break
			}
			// Followers might lag slightly, but should catch up
			if leaderCommit > 0 && node.GetCommitIndex() < leaderCommit-1 {
				allConverged = false
				break
			}
		}

		if allConverged {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	tc.t.Fatalf("Cluster failed to converge within %v", timeout)
}

// PartitionNode isolates a node from the rest of the cluster
func (tc *TestCluster) PartitionNode(index int) {
	tc.t.Helper()

	if index < 0 || index >= len(tc.nodes) {
		tc.t.Fatalf("Invalid node index: %d", index)
	}

	// Disconnect the node from all peers
	tc.nodes[index].DisconnectAll()
}

// HealPartition reconnects a partitioned node to the cluster
func (tc *TestCluster) HealPartition(index int) {
	tc.t.Helper()

	if index < 0 || index >= len(tc.nodes) {
		tc.t.Fatalf("Invalid node index: %d", index)
	}

	// Reconnect the node to all peers
	tc.nodes[index].ReconnectAll()
}

// CreateTempDir creates a temporary directory for testing
func CreateTempDir(t *testing.T, prefix string) string {
	t.Helper()

	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// CreateDataDir creates a data directory structure for a node
func CreateDataDir(t *testing.T, nodeID string) string {
	t.Helper()

	baseDir := CreateTempDir(t, "keyval-test-")

	// Create subdirectories
	subdirs := []string{"wal", "snapshots", "data"}
	for _, subdir := range subdirs {
		dir := filepath.Join(baseDir, subdir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	return baseDir
}
