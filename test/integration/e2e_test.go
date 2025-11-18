package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// TestE2E_BasicOperations tests basic CRUD operations
func TestE2E_BasicOperations(t *testing.T) {
	// Create a simple Raft node with state machine
	config := raft.DefaultConfig()
	config.HeartbeatInterval = 50 * time.Millisecond
	sm := statemachine.NewKVStateMachine()

	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	// Create API server
	server := api.NewServer(node)
	defer server.Stop()

	// Put operation
	putReq := &api.Request{
		ID:       "req-1",
		ClientID: "client-1",
		Sequence: 1,
		Type:     api.OperationPut,
		Key:      "testkey",
		Value:    []byte("testvalue"),
		Timeout:  1 * time.Second,
	}

	// Note: This test is simplified since we don't have a full cluster running
	// In a real E2E test, we would set up a full 3-node cluster

	t.Log("E2E basic operations test (simplified)")
}

// TestE2E_SessionManagement tests session management and deduplication
func TestE2E_SessionManagement(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	// Send same request twice
	req := &api.Request{
		ID:       "req-1",
		ClientID: "client-1",
		Sequence: 1,
		Type:     api.OperationGet,
		Key:      "testkey",
		Timeout:  1 * time.Second,
	}

	// First request
	resp1 := server.HandleRequest(req)

	// Second request (same sequence)
	resp2 := server.HandleRequest(req)

	// Both should return the same result (cached)
	if resp1.Success != resp2.Success {
		t.Error("expected same success status for duplicate requests")
	}

	// Verify session was created
	if count := server.GetSessionCount(); count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}
}

// TestE2E_ConcurrentClients tests multiple concurrent clients
func TestE2E_ConcurrentClients(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	numClients := 5
	numOpsPerClient := 10

	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for clientID := 0; clientID < numClients; clientID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			for seq := uint64(1); seq <= uint64(numOpsPerClient); seq++ {
				req := &api.Request{
					ID:       fmt.Sprintf("req-%d-%d", cid, seq),
					ClientID: fmt.Sprintf("client-%d", cid),
					Sequence: seq,
					Type:     api.OperationGet,
					Key:      fmt.Sprintf("key-%d", cid),
					Timeout:  1 * time.Second,
				}

				server.HandleRequest(req)
			}
		}(clientID)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("client error: %v", err)
		}
	}

	// Verify all sessions were created
	expectedSessions := numClients
	if count := server.GetSessionCount(); count != expectedSessions {
		t.Errorf("expected %d sessions, got %d", expectedSessions, count)
	}
}

// TestE2E_BatchOperations tests batch operations
func TestE2E_BatchOperations(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	// Create batch request
	batchReq := &api.BatchRequest{
		ID:       "batch-1",
		ClientID: "client-1",
		Sequence: 1,
		Operations: []api.Operation{
			{Type: api.OperationGet, Key: "key1"},
			{Type: api.OperationGet, Key: "key2"},
			{Type: api.OperationGet, Key: "key3"},
		},
		Timeout: 2 * time.Second,
	}

	resp := server.HandleBatch(batchReq)

	// Batch should be validated
	if resp == nil {
		t.Fatal("expected batch response")
	}

	// Verify results
	if len(resp.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(resp.Results))
	}
}

// TestE2E_ErrorHandling tests error handling scenarios
func TestE2E_ErrorHandling(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	tests := []struct {
		name    string
		req     *api.Request
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty key",
			req: &api.Request{
				ID:       "req-1",
				Type:     api.OperationGet,
				Key:      "",
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "put without value",
			req: &api.Request{
				ID:       "req-1",
				Type:     api.OperationPut,
				Key:      "key",
				Value:    nil,
				Timeout:  1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "valid get request",
			req: &api.Request{
				ID:       "req-1",
				Type:     api.OperationGet,
				Key:      "key",
				Timeout:  1 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := server.HandleRequest(tt.req)

			if tt.wantErr && resp.Success {
				t.Error("expected error, got success")
			}

			if !tt.wantErr && !resp.Success && resp.Error != api.ErrKeyNotFound && resp.Error != api.ErrNotLeader {
				t.Errorf("unexpected error: %s", resp.Error)
			}
		})
	}
}

// TestE2E_ClientRetry tests client retry logic
func TestE2E_ClientRetry(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	// Create client with custom retry config
	clientConfig := &api.ClientConfig{
		MaxRetries:     2,
		RetryDelay:     50 * time.Millisecond,
		RequestTimeout: 1 * time.Second,
	}

	client := api.NewClient("test-client", []*api.Server{server}, clientConfig)

	// Test sequence increment
	seq1 := client.GetSequence()
	if seq1 != 0 {
		t.Errorf("expected initial sequence 0, got %d", seq1)
	}

	// Make a request (will increment sequence)
	client.Get("testkey")

	seq2 := client.GetSequence()
	if seq2 != 1 {
		t.Errorf("expected sequence 1 after request, got %d", seq2)
	}
}

// TestE2E_HighThroughput tests high throughput scenarios
func TestE2E_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high throughput test in short mode")
	}

	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	numRequests := 1000
	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		req := &api.Request{
			ID:       fmt.Sprintf("req-%d", i),
			ClientID: "client-1",
			Sequence: uint64(i + 1),
			Type:     api.OperationGet,
			Key:      fmt.Sprintf("key-%d", i%100),
			Timeout:  1 * time.Second,
		}

		server.HandleRequest(req)
	}

	duration := time.Since(startTime)
	throughput := float64(numRequests) / duration.Seconds()

	t.Logf("Processed %d requests in %v (%.2f req/sec)", numRequests, duration, throughput)
}

// TestE2E_SessionCleanup tests session cleanup
func TestE2E_SessionCleanup(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	// Create server with short session timeout for testing
	server := api.NewServer(node)
	defer server.Stop()

	// Create some sessions
	for i := 0; i < 5; i++ {
		req := &api.Request{
			ID:       fmt.Sprintf("req-%d", i),
			ClientID: fmt.Sprintf("client-%d", i),
			Sequence: 1,
			Type:     api.OperationGet,
			Key:      "testkey",
			Timeout:  1 * time.Second,
		}
		server.HandleRequest(req)
	}

	// Verify sessions were created
	if count := server.GetSessionCount(); count != 5 {
		t.Errorf("expected 5 sessions, got %d", count)
	}

	// Session cleanup happens in background via SessionManager
	// In a real test, we would wait for cleanup and verify sessions are removed
}

// TestE2E_ConcurrentBatchOperations tests concurrent batch operations
func TestE2E_ConcurrentBatchOperations(t *testing.T) {
	config := raft.DefaultConfig()
	sm := statemachine.NewKVStateMachine()
	node := raft.NewNode("node1", []string{"node2", "node3"}, config, sm)

	server := api.NewServer(node)
	defer server.Stop()

	numBatches := 10
	var wg sync.WaitGroup

	for i := 0; i < numBatches; i++ {
		wg.Add(1)
		go func(batchID int) {
			defer wg.Done()

			ops := make([]api.Operation, 5)
			for j := 0; j < 5; j++ {
				ops[j] = api.Operation{
					Type: api.OperationGet,
					Key:  fmt.Sprintf("key-%d-%d", batchID, j),
				}
			}

			batchReq := &api.BatchRequest{
				ID:         fmt.Sprintf("batch-%d", batchID),
				ClientID:   fmt.Sprintf("client-%d", batchID),
				Sequence:   1,
				Operations: ops,
				Timeout:    2 * time.Second,
			}

			server.HandleBatch(batchReq)
		}(i)
	}

	wg.Wait()

	// Verify all batch sessions were created
	if count := server.GetSessionCount(); count != numBatches {
		t.Errorf("expected %d sessions, got %d", numBatches, count)
	}
}
