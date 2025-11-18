package testutil

import (
	"math/rand"
	"testing"
	"time"
)

// ChaosScheduler manages chaos events in tests
type ChaosScheduler struct {
	t       *testing.T
	cluster *TestCluster
	rng     *rand.Rand
	stopCh  chan struct{}
}

// NewChaosScheduler creates a new chaos scheduler
func NewChaosScheduler(t *testing.T, cluster *TestCluster) *ChaosScheduler {
	return &ChaosScheduler{
		t:       t,
		cluster: cluster,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stopCh:  make(chan struct{}),
	}
}

// Start begins chaos operations
func (cs *ChaosScheduler) Start(interval time.Duration) {
	go cs.run(interval)
}

// Stop halts chaos operations
func (cs *ChaosScheduler) Stop() {
	close(cs.stopCh)
}

// run executes random chaos events at the given interval
func (cs *ChaosScheduler) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-cs.stopCh:
			return
		case <-ticker.C:
			cs.randomChaosEvent()
		}
	}
}

// randomChaosEvent triggers a random chaos event
func (cs *ChaosScheduler) randomChaosEvent() {
	events := []func(){
		cs.randomNodeStop,
		cs.randomNodeRestart,
		cs.randomPartition,
		cs.randomHealPartition,
	}

	// Pick a random event
	event := events[cs.rng.Intn(len(events))]
	event()
}

// randomNodeStop stops a random node
func (cs *ChaosScheduler) randomNodeStop() {
	index := cs.rng.Intn(cs.cluster.Size())
	cs.t.Logf("Chaos: Stopping node %d", index)
	cs.cluster.StopNode(index)
}

// randomNodeRestart restarts a random stopped node
func (cs *ChaosScheduler) randomNodeRestart() {
	index := cs.rng.Intn(cs.cluster.Size())
	cs.t.Logf("Chaos: Restarting node %d", index)
	cs.cluster.StartNode(index)
}

// randomPartition creates a random network partition
func (cs *ChaosScheduler) randomPartition() {
	index := cs.rng.Intn(cs.cluster.Size())
	cs.t.Logf("Chaos: Partitioning node %d", index)
	cs.cluster.PartitionNode(index)
}

// randomHealPartition heals a random network partition
func (cs *ChaosScheduler) randomHealPartition() {
	index := cs.rng.Intn(cs.cluster.Size())
	cs.t.Logf("Chaos: Healing partition for node %d", index)
	cs.cluster.HealPartition(index)
}

// NetworkPartition represents a network partition scenario
type NetworkPartition struct {
	Group1 []int
	Group2 []int
}

// CreateNetworkPartition creates a network partition with two groups
func CreateNetworkPartition(cluster *TestCluster, group1, group2 []int) {
	// Disconnect nodes in group1 from nodes in group2
	for _, i := range group1 {
		for _, j := range group2 {
			cluster.GetNode(i).DisconnectPeer(cluster.GetNode(j).GetID())
		}
	}

	// Disconnect nodes in group2 from nodes in group1
	for _, i := range group2 {
		for _, j := range group1 {
			cluster.GetNode(i).DisconnectPeer(cluster.GetNode(j).GetID())
		}
	}
}

// HealNetworkPartition heals a network partition
func HealNetworkPartition(cluster *TestCluster, partition NetworkPartition) {
	// Reconnect all nodes
	for _, i := range partition.Group1 {
		cluster.HealPartition(i)
	}
	for _, i := range partition.Group2 {
		cluster.HealPartition(i)
	}
}

// SimulateCrash simulates a node crash by stopping it without cleanup
func SimulateCrash(t *testing.T, cluster *TestCluster, nodeIndex int) {
	t.Helper()
	t.Logf("Simulating crash of node %d", nodeIndex)
	cluster.StopNode(nodeIndex)
}

// SimulateSlowNetwork simulates slow network by introducing delays
func SimulateSlowNetwork(t *testing.T, cluster *TestCluster, delay time.Duration) {
	t.Helper()
	t.Logf("Simulating slow network with %v delay", delay)
	// This would be implemented by adding network delay to RPC calls
	// For now, it's a placeholder
}

// SimulatePacketLoss simulates packet loss in the network
func SimulatePacketLoss(t *testing.T, cluster *TestCluster, lossRate float64) {
	t.Helper()
	t.Logf("Simulating packet loss with rate %.2f", lossRate)
	// This would be implemented by dropping random RPC calls
	// For now, it's a placeholder
}

// RandomWorkload generates random operations on the cluster
type RandomWorkload struct {
	t       *testing.T
	cluster *TestCluster
	rng     *rand.Rand
	stopCh  chan struct{}
}

// NewRandomWorkload creates a new random workload generator
func NewRandomWorkload(t *testing.T, cluster *TestCluster) *RandomWorkload {
	return &RandomWorkload{
		t:       t,
		cluster: cluster,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
		stopCh:  make(chan struct{}),
	}
}

// Start begins generating random operations
func (rw *RandomWorkload) Start(opsPerSecond int) {
	interval := time.Second / time.Duration(opsPerSecond)
	go rw.run(interval)
}

// Stop halts workload generation
func (rw *RandomWorkload) Stop() {
	close(rw.stopCh)
}

// run executes random operations at the given interval
func (rw *RandomWorkload) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rw.stopCh:
			return
		case <-ticker.C:
			rw.randomOperation()
		}
	}
}

// randomOperation performs a random operation
func (rw *RandomWorkload) randomOperation() {
	operations := []string{"put", "get", "delete"}
	op := operations[rw.rng.Intn(len(operations))]

	key := randomKey(rw.rng)

	switch op {
	case "put":
		value := randomValue(rw.rng)
		rw.t.Logf("Workload: PUT %s = %s", key, value)
		// Perform put operation on leader
	case "get":
		rw.t.Logf("Workload: GET %s", key)
		// Perform get operation
	case "delete":
		rw.t.Logf("Workload: DELETE %s", key)
		// Perform delete operation
	}
}

// randomKey generates a random key
func randomKey(rng *rand.Rand) string {
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	return keys[rng.Intn(len(keys))]
}

// randomValue generates a random value
func randomValue(rng *rand.Rand) string {
	values := []string{"value1", "value2", "value3", "value4", "value5"}
	return values[rng.Intn(len(values))]
}
