package api

import (
	"testing"
	"time"
)

func TestClient_RequestIDGeneration(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	// Generate multiple request IDs
	id1 := client.generateRequestID()
	id2 := client.generateRequestID()

	if id1 == id2 {
		t.Error("expected unique request IDs")
	}

	if id1 == "" || id2 == "" {
		t.Error("expected non-empty request IDs")
	}
}

func TestClient_SequenceIncrement(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	if seq := client.GetSequence(); seq != 0 {
		t.Errorf("expected initial sequence 0, got %d", seq)
	}

	// Create requests
	client.newRequest(OperationPut, "key1", []byte("value1"))
	if seq := client.GetSequence(); seq != 1 {
		t.Errorf("expected sequence 1, got %d", seq)
	}

	client.newRequest(OperationGet, "key2", nil)
	if seq := client.GetSequence(); seq != 2 {
		t.Errorf("expected sequence 2, got %d", seq)
	}

	client.newRequest(OperationDelete, "key3", nil)
	if seq := client.GetSequence(); seq != 3 {
		t.Errorf("expected sequence 3, got %d", seq)
	}
}

func TestClient_NewRequest(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	req := client.newRequest(OperationPut, "testkey", []byte("testvalue"))

	if req.ClientID != "test-client" {
		t.Errorf("expected client ID 'test-client', got '%s'", req.ClientID)
	}

	if req.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", req.Sequence)
	}

	if req.Type != OperationPut {
		t.Errorf("expected Put operation, got %v", req.Type)
	}

	if req.Key != "testkey" {
		t.Errorf("expected key 'testkey', got '%s'", req.Key)
	}

	if string(req.Value) != "testvalue" {
		t.Errorf("expected value 'testvalue', got '%s'", string(req.Value))
	}

	if req.ID == "" {
		t.Error("expected non-empty request ID")
	}
}

func TestClient_NewBatchRequest(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	ops := []Operation{
		{Type: OperationPut, Key: "key1", Value: []byte("value1")},
		{Type: OperationGet, Key: "key2"},
	}

	batchReq := client.newBatchRequest(ops)

	if batchReq.ClientID != "test-client" {
		t.Errorf("expected client ID 'test-client', got '%s'", batchReq.ClientID)
	}

	if batchReq.Sequence != 1 {
		t.Errorf("expected sequence 1, got %d", batchReq.Sequence)
	}

	if len(batchReq.Operations) != 2 {
		t.Errorf("expected 2 operations, got %d", len(batchReq.Operations))
	}

	if batchReq.ID == "" {
		t.Error("expected non-empty request ID")
	}
}

func TestClient_SetRequestTimeout(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	// Default timeout
	req := client.newRequest(OperationGet, "key", nil)
	if req.Timeout != 5*time.Second {
		t.Errorf("expected default timeout 5s, got %v", req.Timeout)
	}

	// Set custom timeout
	client.SetRequestTimeout(1 * time.Second)

	req = client.newRequest(OperationGet, "key", nil)
	if req.Timeout != 1*time.Second {
		t.Errorf("expected timeout 1s, got %v", req.Timeout)
	}
}

func TestClient_SetRetryConfig(t *testing.T) {
	config := &ClientConfig{
		MaxRetries:     5,
		RetryDelay:     200 * time.Millisecond,
		RequestTimeout: 3 * time.Second,
	}

	client := NewClient("test-client", nil, config)

	if client.maxRetries != 5 {
		t.Errorf("expected max retries 5, got %d", client.maxRetries)
	}

	if client.retryDelay != 200*time.Millisecond {
		t.Errorf("expected retry delay 200ms, got %v", client.retryDelay)
	}

	// Change retry config
	client.SetRetryConfig(3, 100*time.Millisecond)

	if client.maxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", client.maxRetries)
	}

	if client.retryDelay != 100*time.Millisecond {
		t.Errorf("expected retry delay 100ms, got %v", client.retryDelay)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	if config.MaxRetries != 3 {
		t.Errorf("expected max retries 3, got %d", config.MaxRetries)
	}

	if config.RetryDelay != 100*time.Millisecond {
		t.Errorf("expected retry delay 100ms, got %v", config.RetryDelay)
	}

	if config.RequestTimeout != 5*time.Second {
		t.Errorf("expected request timeout 5s, got %v", config.RequestTimeout)
	}
}

func TestClient_ConcurrentRequests(t *testing.T) {
	client := NewClient("test-client", nil, nil)

	numGoroutines := 10
	numRequests := 100
	done := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numRequests; j++ {
				client.newRequest(OperationPut, "key", []byte("value"))
			}
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify sequence number
	expectedSeq := uint64(numGoroutines * numRequests)
	if seq := client.GetSequence(); seq != expectedSeq {
		t.Errorf("expected sequence %d, got %d", expectedSeq, seq)
	}
}
