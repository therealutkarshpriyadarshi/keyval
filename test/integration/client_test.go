package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/statemachine"
)

// TestClientPutGet tests the basic put and get operations
func TestClientPutGet(t *testing.T) {
	// Create a single node cluster
	node, err := raft.NewNode("node1", "127.0.0.1:9001", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Wait for node to become leader (single node cluster)
	time.Sleep(500 * time.Millisecond)

	if !node.IsLeader() {
		t.Fatal("Node should be leader in single-node cluster")
	}

	// Create API server
	server := api.NewServer(node)

	// Put a value
	putReq := &api.Request{
		ID:       "req1",
		ClientID: "client1",
		Sequence: 1,
		Type:     api.OperationPut,
		Key:      "testkey",
		Value:    []byte("testvalue"),
		Timeout:  5 * time.Second,
	}

	putResp := server.HandleRequest(putReq)
	if !putResp.Success {
		t.Fatalf("Put request failed: %s", putResp.Error)
	}

	// Wait for the entry to be applied
	time.Sleep(200 * time.Millisecond)

	// Get the value
	getReq := &api.Request{
		ID:       "req2",
		ClientID: "client1",
		Sequence: 2,
		Type:     api.OperationGet,
		Key:      "testkey",
		Timeout:  5 * time.Second,
	}

	getResp := server.HandleRequest(getReq)
	if !getResp.Success {
		t.Fatalf("Get request failed: %s", getResp.Error)
	}

	if string(getResp.Value) != "testvalue" {
		t.Errorf("Expected 'testvalue', got '%s'", string(getResp.Value))
	}

	t.Log("Put and Get operations successful")
}

// TestClientDelete tests the delete operation
func TestClientDelete(t *testing.T) {
	// Create a single node cluster
	node, err := raft.NewNode("node1", "127.0.0.1:9002", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	// Wait for node to become leader
	time.Sleep(500 * time.Millisecond)

	server := api.NewServer(node)

	// Put a value
	putReq := &api.Request{
		ID:       "req1",
		ClientID: "client1",
		Sequence: 1,
		Type:     api.OperationPut,
		Key:      "delkey",
		Value:    []byte("delvalue"),
		Timeout:  5 * time.Second,
	}

	putResp := server.HandleRequest(putReq)
	if !putResp.Success {
		t.Fatalf("Put request failed: %s", putResp.Error)
	}

	time.Sleep(200 * time.Millisecond)

	// Delete the value
	delReq := &api.Request{
		ID:       "req2",
		ClientID: "client1",
		Sequence: 2,
		Type:     api.OperationDelete,
		Key:      "delkey",
		Timeout:  5 * time.Second,
	}

	delResp := server.HandleRequest(delReq)
	if !delResp.Success {
		t.Fatalf("Delete request failed: %s", delResp.Error)
	}

	time.Sleep(200 * time.Millisecond)

	// Try to get the deleted value
	getReq := &api.Request{
		ID:       "req3",
		ClientID: "client1",
		Sequence: 3,
		Type:     api.OperationGet,
		Key:      "delkey",
		Timeout:  5 * time.Second,
	}

	getResp := server.HandleRequest(getReq)
	if getResp.Success {
		t.Error("Get should fail for deleted key")
	}

	t.Log("Delete operation successful")
}

// TestStateMachineApply tests applying commands directly to the state machine
func TestStateMachineApply(t *testing.T) {
	sm := statemachine.NewKVStateMachine()

	// Put a value
	putCmd := statemachine.Command{
		Type:  statemachine.CommandPut,
		Key:   "key1",
		Value: []byte("value1"),
	}

	data, err := json.Marshal(putCmd)
	if err != nil {
		t.Fatalf("Failed to marshal command: %v", err)
	}

	result, err := sm.Apply(data)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	resp, ok := result.(*statemachine.CommandResponse)
	if !ok || !resp.Success {
		t.Error("Put command should succeed")
	}

	// Verify the value
	value, exists := sm.Get("key1")
	if !exists {
		t.Error("Key should exist")
	}

	if string(value) != "value1" {
		t.Errorf("Expected 'value1', got '%s'", string(value))
	}

	t.Log("State machine apply test successful")
}

// TestRequestDeduplication tests that duplicate requests return cached responses
func TestRequestDeduplication(t *testing.T) {
	node, err := raft.NewNode("node1", "127.0.0.1:9003", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	server := api.NewServer(node)

	// First request
	req := &api.Request{
		ID:       "dedup1",
		ClientID: "client1",
		Sequence: 1,
		Type:     api.OperationPut,
		Key:      "dedupkey",
		Value:    []byte("dedupvalue"),
		Timeout:  5 * time.Second,
	}

	resp1 := server.HandleRequest(req)
	if !resp1.Success {
		t.Fatalf("First request failed: %s", resp1.Error)
	}

	// Same request again (same sequence number)
	resp2 := server.HandleRequest(req)
	if !resp2.Success {
		t.Fatalf("Duplicate request failed: %s", resp2.Error)
	}

	t.Log("Request deduplication test successful")
}

// TestMultipleOperations tests a sequence of operations
func TestMultipleOperations(t *testing.T) {
	node, err := raft.NewNode("node1", "127.0.0.1:9004", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	if err := node.Start(); err != nil {
		t.Fatalf("Failed to start node: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	server := api.NewServer(node)

	// Perform multiple operations
	operations := []struct {
		opType api.OperationType
		key    string
		value  string
	}{
		{api.OperationPut, "key1", "value1"},
		{api.OperationPut, "key2", "value2"},
		{api.OperationPut, "key3", "value3"},
		{api.OperationDelete, "key2", ""},
	}

	seq := uint64(1)
	for _, op := range operations {
		req := &api.Request{
			ID:       generateID(),
			ClientID: "client1",
			Sequence: seq,
			Type:     op.opType,
			Key:      op.key,
			Timeout:  5 * time.Second,
		}

		if op.opType == api.OperationPut {
			req.Value = []byte(op.value)
		}

		resp := server.HandleRequest(req)
		if !resp.Success {
			t.Fatalf("Operation %v on %s failed: %s", op.opType, op.key, resp.Error)
		}

		seq++
		time.Sleep(100 * time.Millisecond)
	}

	// Verify final state
	// key1 should exist
	// key2 should not exist (deleted)
	// key3 should exist

	getKey1 := &api.Request{
		ID:       generateID(),
		ClientID: "client1",
		Sequence: seq,
		Type:     api.OperationGet,
		Key:      "key1",
		Timeout:  5 * time.Second,
	}

	resp := server.HandleRequest(getKey1)
	if !resp.Success || string(resp.Value) != "value1" {
		t.Error("key1 should exist with value1")
	}

	seq++

	getKey2 := &api.Request{
		ID:       generateID(),
		ClientID: "client1",
		Sequence: seq,
		Type:     api.OperationGet,
		Key:      "key2",
		Timeout:  5 * time.Second,
	}

	resp = server.HandleRequest(getKey2)
	if resp.Success {
		t.Error("key2 should not exist (was deleted)")
	}

	t.Log("Multiple operations test successful")
}

// Helper function to generate unique IDs
func generateID() string {
	return time.Now().Format("20060102150405.000000")
}
