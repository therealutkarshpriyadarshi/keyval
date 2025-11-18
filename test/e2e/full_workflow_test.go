// +build e2e

package e2e

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
	"github.com/therealutkarshpriyadarshi/keyval/test/testutil"
)

func TestE2E_FullWorkflow(t *testing.T) {
	// Create a 3-node cluster
	config := testutil.DefaultClusterConfig()
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	// Wait for leader
	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	t.Logf("Leader elected: %s", leader.GetID())

	// Create API server connected to the leader
	server := api.NewServer(leader)
	defer server.Stop()

	// 1. Test basic PUT operation
	t.Run("BasicPut", func(t *testing.T) {
		req := &api.Request{
			ID:       "req-1",
			ClientID: "client-1",
			Sequence: 1,
			Type:     api.OperationPut,
			Key:      "user:1",
			Value:    []byte("John Doe"),
			Timeout:  2 * time.Second,
		}

		resp := server.HandleRequest(req)
		testutil.AssertTrue(t, resp.Success, "PUT should succeed")
		t.Logf("PUT successful: %s = %s", req.Key, req.Value)
	})

	// 2. Test GET operation
	t.Run("BasicGet", func(t *testing.T) {
		req := &api.Request{
			ID:       "req-2",
			ClientID: "client-1",
			Sequence: 2,
			Type:     api.OperationGet,
			Key:      "user:1",
			Timeout:  2 * time.Second,
		}

		resp := server.HandleRequest(req)
		testutil.AssertTrue(t, resp.Success, "GET should succeed")
		testutil.AssertEqual(t, "John Doe", string(resp.Value), "Value should match")
		t.Logf("GET successful: %s = %s", req.Key, resp.Value)
	})

	// 3. Test DELETE operation
	t.Run("BasicDelete", func(t *testing.T) {
		req := &api.Request{
			ID:       "req-3",
			ClientID: "client-1",
			Sequence: 3,
			Type:     api.OperationDelete,
			Key:      "user:1",
			Timeout:  2 * time.Second,
		}

		resp := server.HandleRequest(req)
		testutil.AssertTrue(t, resp.Success, "DELETE should succeed")
		t.Logf("DELETE successful: %s", req.Key)

		// Verify key is deleted
		getReq := &api.Request{
			ID:       "req-4",
			ClientID: "client-1",
			Sequence: 4,
			Type:     api.OperationGet,
			Key:      "user:1",
			Timeout:  2 * time.Second,
		}

		getResp := server.HandleRequest(getReq)
		testutil.AssertFalse(t, getResp.Success, "GET should fail for deleted key")
		testutil.AssertEqual(t, api.ErrKeyNotFound, getResp.Error, "Should return key not found")
	})
}

func TestE2E_ConcurrentOperations(t *testing.T) {
	// Create cluster
	config := testutil.FastClusterConfig()
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	server := api.NewServer(leader)
	defer server.Stop()

	// Run concurrent operations
	numClients := 10
	opsPerClient := 50

	var wg sync.WaitGroup
	errors := make(chan error, numClients*opsPerClient)

	for clientID := 0; clientID < numClients; clientID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			for seq := 1; seq <= opsPerClient; seq++ {
				key := fmt.Sprintf("client-%d-key-%d", cid, seq%10)
				value := fmt.Sprintf("value-%d", seq)

				// PUT
				putReq := &api.Request{
					ID:       fmt.Sprintf("req-%d-%d", cid, seq),
					ClientID: fmt.Sprintf("client-%d", cid),
					Sequence: uint64(seq),
					Type:     api.OperationPut,
					Key:      key,
					Value:    []byte(value),
					Timeout:  2 * time.Second,
				}

				putResp := server.HandleRequest(putReq)
				if !putResp.Success {
					errors <- fmt.Errorf("PUT failed: %s", putResp.Error)
					continue
				}

				// GET
				getReq := &api.Request{
					ID:       fmt.Sprintf("req-%d-%d-get", cid, seq),
					ClientID: fmt.Sprintf("client-%d", cid),
					Sequence: uint64(seq + 1),
					Type:     api.OperationGet,
					Key:      key,
					Timeout:  2 * time.Second,
				}

				getResp := server.HandleRequest(getReq)
				if !getResp.Success {
					errors <- fmt.Errorf("GET failed: %s", getResp.Error)
				}
			}
		}(clientID)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Encountered %d errors during concurrent operations", errorCount)
	}

	t.Logf("Successfully completed %d concurrent operations", numClients*opsPerClient)
}

func TestE2E_LeaderFailoverDuringOperations(t *testing.T) {
	// Create cluster
	config := testutil.FastClusterConfig()
	config.Size = 3
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	initialLeaderID := leader.GetID()

	server := api.NewServer(leader)
	defer server.Stop()

	// Start background operations
	stopCh := make(chan struct{})
	errorsCh := make(chan error, 100)

	go func() {
		seq := uint64(1)
		for {
			select {
			case <-stopCh:
				return
			default:
				req := &api.Request{
					ID:       fmt.Sprintf("req-%d", seq),
					ClientID: "bg-client",
					Sequence: seq,
					Type:     api.OperationPut,
					Key:      fmt.Sprintf("key-%d", seq%10),
					Value:    []byte(fmt.Sprintf("value-%d", seq)),
					Timeout:  2 * time.Second,
				}

				resp := server.HandleRequest(req)
				if !resp.Success && resp.Error != api.ErrNotLeader {
					errorsCh <- fmt.Errorf("operation %d failed: %s", seq, resp.Error)
				}

				seq++
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	// Let some operations run
	time.Sleep(1 * time.Second)

	// Simulate leader failure
	t.Logf("Simulating leader failure: %s", initialLeaderID)
	for i, node := range cluster.GetNodes() {
		if node.GetID() == initialLeaderID {
			cluster.StopNode(i)
			break
		}
	}

	// Wait for new leader
	time.Sleep(1 * time.Second)
	newLeader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	t.Logf("New leader elected: %s", newLeader.GetID())

	// Continue operations for a bit longer
	time.Sleep(2 * time.Second)

	// Stop background operations
	close(stopCh)
	close(errorsCh)

	// Check for errors
	errorCount := 0
	for err := range errorsCh {
		t.Logf("Error: %v", err)
		errorCount++
	}

	// Some errors are acceptable during failover
	if errorCount > 20 {
		t.Fatalf("Too many errors during leader failover: %d", errorCount)
	}

	t.Logf("Completed failover test with %d errors", errorCount)
}

func TestE2E_LongRunningStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running stress test in short mode")
	}

	// Create cluster
	config := testutil.DefaultClusterConfig()
	config.Size = 5
	cluster := testutil.NewTestCluster(t, config)
	defer cluster.Stop()

	cluster.Start()

	leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)
	server := api.NewServer(leader)
	defer server.Stop()

	// Run stress test for 1 minute
	duration := 1 * time.Minute
	deadline := time.Now().Add(duration)

	numClients := 20
	var wg sync.WaitGroup
	totalOps := 0
	var opsLock sync.Mutex

	for clientID := 0; clientID < numClients; clientID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			seq := uint64(1)
			clientOps := 0

			for time.Now().Before(deadline) {
				key := fmt.Sprintf("stress-key-%d", seq%100)
				value := fmt.Sprintf("stress-value-%d", seq)

				req := &api.Request{
					ID:       fmt.Sprintf("stress-req-%d-%d", cid, seq),
					ClientID: fmt.Sprintf("stress-client-%d", cid),
					Sequence: seq,
					Type:     api.OperationPut,
					Key:      key,
					Value:    []byte(value),
					Timeout:  2 * time.Second,
				}

				server.HandleRequest(req)
				seq++
				clientOps++

				time.Sleep(10 * time.Millisecond)
			}

			opsLock.Lock()
			totalOps += clientOps
			opsLock.Unlock()
		}(clientID)
	}

	wg.Wait()

	throughput := float64(totalOps) / duration.Seconds()
	t.Logf("Stress test completed: %d total operations, %.2f ops/sec", totalOps, throughput)

	// Verify cluster is still healthy
	testutil.AssertSingleLeader(t, cluster)
	testutil.AssertConvergence(t, cluster, 10*time.Second)
}
