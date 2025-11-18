// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/test/testutil"
)

func TestRaftCluster_BasicElection(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 3

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	leader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	t.Logf("Leader elected: %s", leader.GetID())

	// Assert single leader
	testutil.AssertSingleLeader(t, cluster)

	// All nodes should agree on term
	term := leader.GetCurrentTerm()
	for _, node := range cluster.GetNodes() {
		testutil.AssertTrue(t, node.GetCurrentTerm() >= term,
			"All nodes should be in current or higher term")
	}
}

func TestRaftCluster_LeaderStability(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 3

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for initial leader
	initialLeader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	initialTerm := initialLeader.GetCurrentTerm()
	initialID := initialLeader.GetID()

	t.Logf("Initial leader: %s in term %d", initialID, initialTerm)

	// Wait and ensure leadership doesn't change
	time.Sleep(2 * time.Second)

	currentLeader := testutil.AssertSingleLeader(t, cluster)
	testutil.AssertEqual(t, initialID, currentLeader.GetID(),
		"Leader should remain stable")
	testutil.AssertEqual(t, initialTerm, currentLeader.GetCurrentTerm(),
		"Term should remain stable")
}

func TestRaftCluster_LeaderFailover(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 3

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for initial leader
	initialLeader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	initialLeaderID := initialLeader.GetID()
	initialTerm := initialLeader.GetCurrentTerm()

	t.Logf("Initial leader: %s in term %d", initialLeaderID, initialTerm)

	// Find the leader's index
	leaderIndex := -1
	for i, node := range cluster.GetNodes() {
		if node.GetID() == initialLeaderID {
			leaderIndex = i
			break
		}
	}

	// Stop the leader
	t.Logf("Stopping leader node %d", leaderIndex)
	cluster.StopNode(leaderIndex)

	// Wait for new leader election
	time.Sleep(500 * time.Millisecond)
	newLeader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)

	// New leader should be different
	testutil.AssertNotEqual(t, initialLeaderID, newLeader.GetID(),
		"New leader should be elected")

	// New term should be higher
	testutil.AssertTrue(t, newLeader.GetCurrentTerm() > initialTerm,
		"New leader should have higher term")

	t.Logf("New leader elected: %s in term %d", newLeader.GetID(), newLeader.GetCurrentTerm())
}

func TestRaftCluster_MinorityFailure(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 5

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	leader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	t.Logf("Initial leader: %s", leader.GetID())

	// Stop minority of nodes (2 out of 5)
	cluster.StopNode(1)
	cluster.StopNode(2)
	t.Log("Stopped 2 nodes (minority)")

	// Cluster should still elect/maintain a leader
	time.Sleep(500 * time.Millisecond)
	currentLeader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	testutil.AssertSingleLeader(t, cluster)

	t.Logf("Leader maintained: %s", currentLeader.GetID())
}

func TestRaftCluster_MajorityFailure(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 5

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	leader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	t.Logf("Initial leader: %s", leader.GetID())

	// Stop majority of nodes (3 out of 5)
	cluster.StopNode(1)
	cluster.StopNode(2)
	cluster.StopNode(3)
	t.Log("Stopped 3 nodes (majority)")

	// Give time for cluster to realize majority is down
	time.Sleep(1 * time.Second)

	// Should not have a leader anymore (no quorum)
	// The existing leader should step down
	testutil.AssertEventually(t, func() bool {
		return cluster.CountLeaders() == 0
	}, 3*time.Second, "No leader should exist without quorum")
}

func TestRaftCluster_NetworkPartition(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 5

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	initialLeader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	t.Logf("Initial leader: %s", initialLeader.GetID())

	// Create partition: [0,1] vs [2,3,4]
	partition := testutil.NetworkPartition{
		Group1: []int{0, 1},
		Group2: []int{2, 3, 4},
	}

	t.Log("Creating network partition: [0,1] vs [2,3,4]")
	testutil.CreateNetworkPartition(cluster, partition.Group1, partition.Group2)

	// Wait for partition to take effect
	time.Sleep(1 * time.Second)

	// Majority partition (2,3,4) should elect a leader
	// Minority partition (0,1) should not
	testutil.AssertEventually(t, func() bool {
		// Check majority partition has a leader
		majorityLeaderCount := 0
		for _, i := range partition.Group2 {
			if cluster.GetNode(i).IsLeader() {
				majorityLeaderCount++
			}
		}

		// Check minority partition has no leader
		minorityLeaderCount := 0
		for _, i := range partition.Group1 {
			if cluster.GetNode(i).IsLeader() {
				minorityLeaderCount++
			}
		}

		return majorityLeaderCount == 1 && minorityLeaderCount == 0
	}, 5*time.Second, "Majority partition should have leader, minority should not")

	// Heal partition
	t.Log("Healing network partition")
	testutil.HealNetworkPartition(cluster, partition)

	// Wait for cluster to converge
	time.Sleep(1 * time.Second)
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 3*time.Second)

	t.Log("Cluster converged after healing partition")
}

func TestRaftCluster_Recovery(t *testing.T) {
	config := testutil.FastClusterConfig()
	config.Size = 3

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	leader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	t.Logf("Initial leader: %s", leader.GetID())

	// Stop all nodes
	t.Log("Stopping all nodes")
	cluster.Stop()

	// Restart all nodes
	time.Sleep(500 * time.Millisecond)
	t.Log("Restarting all nodes")
	cluster.Start()

	// Cluster should recover and elect a leader
	newLeader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	testutil.AssertSingleLeader(t, cluster)

	t.Logf("Cluster recovered with leader: %s", newLeader.GetID())
}

func TestRaftCluster_ConcurrentFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent failures test in short mode")
	}

	config := testutil.FastClusterConfig()
	config.Size = 5

	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader election
	testutil.AssertLeaderElected(t, cluster, 3*time.Second)

	// Simulate cascading failures
	t.Log("Simulating cascading failures")

	cluster.StopNode(0)
	time.Sleep(200 * time.Millisecond)

	cluster.StopNode(1)
	time.Sleep(200 * time.Millisecond)

	// Should still have quorum (3 out of 5)
	testutil.AssertLeaderElected(t, cluster, 3*time.Second)

	// Restart nodes one by one
	t.Log("Recovering nodes")
	cluster.StartNode(0)
	time.Sleep(200 * time.Millisecond)

	cluster.StartNode(1)
	time.Sleep(500 * time.Millisecond)

	// Cluster should converge
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 5*time.Second)

	t.Log("Cluster recovered from cascading failures")
}
