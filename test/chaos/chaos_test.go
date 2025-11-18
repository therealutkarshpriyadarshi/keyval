// +build chaos

package chaos

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
	"github.com/therealutkarshpriyadarshi/keyval/test/testutil"
)

// TestChaos_RandomFailures tests the cluster under random node failures
func TestChaos_RandomFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	// Create a 5-node cluster for better fault tolerance
	config := testutil.DefaultClusterConfig()
	config.Size = 5
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for initial leader
	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	t.Logf("Initial leader: %s", leader.GetID())

	// Start chaos scheduler
	chaos := testutil.NewChaosScheduler(t, cluster)
	chaos.Start(2 * time.Second)
	defer chaos.Stop()

	// Run workload during chaos
	workload := testutil.NewRandomWorkload(t, cluster)
	workload.Start(10) // 10 ops/second
	defer workload.Stop()

	// Let chaos run for 30 seconds
	t.Log("Running chaos test for 30 seconds...")
	time.Sleep(30 * time.Second)

	// Stop chaos and workload
	chaos.Stop()
	workload.Stop()

	// Restart all nodes to ensure they can recover
	t.Log("Ensuring all nodes are running...")
	for i := 0; i < cluster.Size(); i++ {
		cluster.StartNode(i)
	}

	// Wait for cluster to stabilize
	time.Sleep(2 * time.Second)

	// Verify cluster health
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 10*time.Second)

	t.Log("Chaos test completed successfully")
}

// TestChaos_NetworkPartitions tests various network partition scenarios
func TestChaos_NetworkPartitions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	scenarios := []struct {
		name   string
		size   int
		group1 []int
		group2 []int
	}{
		{
			name:   "Split 3-node cluster (1 vs 2)",
			size:   3,
			group1: []int{0},
			group2: []int{1, 2},
		},
		{
			name:   "Split 5-node cluster (2 vs 3)",
			size:   5,
			group1: []int{0, 1},
			group2: []int{2, 3, 4},
		},
		{
			name:   "Split 5-node cluster evenly (2 vs 2, with 1 isolated)",
			size:   5,
			group1: []int{0, 1},
			group2: []int{2, 3},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			config := testutil.FastClusterConfig()
			config.Size = scenario.size
			cluster := testutil.NewTestCluster(t, config)
			defer cluster.Stop()

			cluster.Start()

			// Wait for leader
			initialLeader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
			t.Logf("Initial leader: %s", initialLeader.GetID())

			// Create partition
			t.Logf("Creating partition: %v vs %v", scenario.group1, scenario.group2)
			partition := testutil.NetworkPartition{
				Group1: scenario.group1,
				Group2: scenario.group2,
			}
			testutil.CreateNetworkPartition(cluster, partition.Group1, partition.Group2)

			// Let partition exist for a while
			time.Sleep(3 * time.Second)

			// Heal partition
			t.Log("Healing partition")
			testutil.HealNetworkPartition(cluster, partition)

			// Wait for convergence
			time.Sleep(2 * time.Second)
			testutil.AssertSingleLeader(t, cluster)
			testutil.AssertConvergence(t, cluster, 5*time.Second)

			t.Log("Partition healed successfully")
		})
	}
}

// TestChaos_CascadingFailures tests cascading node failures
func TestChaos_CascadingFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	config := testutil.DefaultClusterConfig()
	config.Size = 7 // Larger cluster for more resilience
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader
	testutil.AssertLeaderElected(t, cluster, 5*time.Second)

	// Simulate cascading failures
	t.Log("Starting cascading failures...")

	// Fail nodes one by one
	for i := 0; i < 3; i++ {
		t.Logf("Failing node %d", i)
		cluster.StopNode(i)
		time.Sleep(500 * time.Millisecond)

		// Should still have quorum
		testutil.AssertLeaderElected(t, cluster, 3*time.Second)
	}

	t.Log("Three nodes failed, cluster still operational")

	// Start recovering nodes
	for i := 0; i < 3; i++ {
		t.Logf("Recovering node %d", i)
		cluster.StartNode(i)
		time.Sleep(500 * time.Millisecond)
	}

	// Wait for full convergence
	time.Sleep(2 * time.Second)
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 10*time.Second)

	t.Log("Cluster recovered from cascading failures")
}

// TestChaos_StressWithFailures tests high load with random failures
func TestChaos_StressWithFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	config := testutil.DefaultClusterConfig()
	config.Size = 5
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	server := api.NewServer(leader)
	defer server.Stop()

	// Start high-load workload
	numClients := 20
	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	successCount := 0
	errorCount := 0
	var statsLock sync.Mutex

	for clientID := 0; clientID < numClients; clientID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			seq := uint64(1)
			for {
				select {
				case <-stopCh:
					return
				default:
					req := &api.Request{
						ID:       fmt.Sprintf("req-%d-%d", cid, seq),
						ClientID: fmt.Sprintf("client-%d", cid),
						Sequence: seq,
						Type:     api.OperationPut,
						Key:      fmt.Sprintf("key-%d", seq%100),
						Value:    []byte(fmt.Sprintf("value-%d", seq)),
						Timeout:  2 * time.Second,
					}

					resp := server.HandleRequest(req)

					statsLock.Lock()
					if resp.Success {
						successCount++
					} else {
						errorCount++
					}
					statsLock.Unlock()

					seq++
					time.Sleep(20 * time.Millisecond)
				}
			}
		}(clientID)
	}

	// Run chaos in background
	go func() {
		for i := 0; i < 5; i++ {
			time.Sleep(2 * time.Second)

			// Random node failure
			nodeIdx := i % cluster.Size()
			t.Logf("Chaos: Stopping node %d", nodeIdx)
			cluster.StopNode(nodeIdx)

			time.Sleep(2 * time.Second)

			// Recover node
			t.Logf("Chaos: Restarting node %d", nodeIdx)
			cluster.StartNode(nodeIdx)
		}
	}()

	// Run test for 20 seconds
	time.Sleep(20 * time.Second)

	// Stop workload
	close(stopCh)
	wg.Wait()

	statsLock.Lock()
	totalOps := successCount + errorCount
	successRate := float64(successCount) / float64(totalOps) * 100
	statsLock.Unlock()

	t.Logf("Completed: %d total ops, %d successful (%.2f%%), %d errors",
		totalOps, successCount, successRate, errorCount)

	// Success rate should be reasonably high even with failures
	if successRate < 80.0 {
		t.Fatalf("Success rate too low: %.2f%%", successRate)
	}

	// Ensure cluster is healthy at the end
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 10*time.Second)
}

// TestChaos_SimulateRealWorldScenarios simulates real-world failure patterns
func TestChaos_SimulateRealWorldScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	scenarios := []struct {
		name     string
		scenario func(t *testing.T, cluster *testutil.TestCluster)
	}{
		{
			name: "Rolling restart (like deployment)",
			scenario: func(t *testing.T, cluster *testutil.TestCluster) {
				for i := 0; i < cluster.Size(); i++ {
					t.Logf("Restarting node %d", i)
					cluster.StopNode(i)
					time.Sleep(500 * time.Millisecond)
					cluster.StartNode(i)
					time.Sleep(1 * time.Second)
				}
			},
		},
		{
			name: "Data center failure (half cluster down)",
			scenario: func(t *testing.T, cluster *testutil.TestCluster) {
				half := cluster.Size() / 2
				t.Logf("Simulating data center failure: stopping %d nodes", half)

				for i := 0; i < half; i++ {
					cluster.StopNode(i)
				}

				time.Sleep(3 * time.Second)

				t.Log("Data center recovered")
				for i := 0; i < half; i++ {
					cluster.StartNode(i)
				}
			},
		},
		{
			name: "Flapping node (repeatedly fails and recovers)",
			scenario: func(t *testing.T, cluster *testutil.TestCluster) {
				nodeIdx := 0
				for i := 0; i < 5; i++ {
					t.Logf("Flap iteration %d: stopping node %d", i+1, nodeIdx)
					cluster.StopNode(nodeIdx)
					time.Sleep(500 * time.Millisecond)

					t.Logf("Flap iteration %d: starting node %d", i+1, nodeIdx)
					cluster.StartNode(nodeIdx)
					time.Sleep(500 * time.Millisecond)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			config := testutil.DefaultClusterConfig()
			config.Size = 5
			cluster := testutil.NewTestCluster(t, config)
			defer cluster.Stop()

			cluster.Start()

			// Wait for initial leader
			testutil.AssertLeaderElected(t, cluster, 5*time.Second)

			// Run scenario
			scenario.scenario(t, cluster)

			// Verify cluster recovers
			time.Sleep(2 * time.Second)
			testutil.AssertSingleLeader(t, cluster)
			testutil.AssertConvergence(t, cluster, 10*time.Second)

			t.Logf("Scenario '%s' completed successfully", scenario.name)
		})
	}
}
