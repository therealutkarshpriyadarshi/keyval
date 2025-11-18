package api

import (
	"testing"
	"time"
)

func TestBatchRequest_Validation(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		req     *BatchRequest
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty operations",
			req: &BatchRequest{
				ID:         "req-1",
				Operations: []Operation{},
			},
			wantErr: true,
		},
		{
			name: "valid single operation",
			req: &BatchRequest{
				ID: "req-1",
				Operations: []Operation{
					{Type: OperationPut, Key: "key1", Value: []byte("value1")},
				},
			},
			wantErr: false,
		},
		{
			name: "valid multiple operations",
			req: &BatchRequest{
				ID: "req-1",
				Operations: []Operation{
					{Type: OperationPut, Key: "key1", Value: []byte("value1")},
					{Type: OperationPut, Key: "key2", Value: []byte("value2")},
					{Type: OperationGet, Key: "key3"},
					{Type: OperationDelete, Key: "key4"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty key",
			req: &BatchRequest{
				ID: "req-1",
				Operations: []Operation{
					{Type: OperationPut, Key: "", Value: []byte("value1")},
				},
			},
			wantErr: true,
		},
		{
			name: "put without value",
			req: &BatchRequest{
				ID: "req-1",
				Operations: []Operation{
					{Type: OperationPut, Key: "key1", Value: nil},
				},
			},
			wantErr: true,
		},
		{
			name: "too many operations",
			req: &BatchRequest{
				ID:         "req-1",
				Operations: make([]Operation, 1001),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.validateBatchRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBatchRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOperation_Types(t *testing.T) {
	ops := []Operation{
		{Type: OperationPut, Key: "key1", Value: []byte("value1")},
		{Type: OperationGet, Key: "key2"},
		{Type: OperationDelete, Key: "key3"},
	}

	// Verify operation properties
	if ops[0].Type != OperationPut {
		t.Error("expected Put operation")
	}

	if ops[1].Type != OperationGet {
		t.Error("expected Get operation")
	}

	if ops[2].Type != OperationDelete {
		t.Error("expected Delete operation")
	}
}

func TestBatchResponse_Structure(t *testing.T) {
	resp := &BatchResponse{
		Success: true,
		Results: []OperationResult{
			{Success: true, Value: []byte("value1")},
			{Success: true, Value: []byte("value2")},
			{Success: false, Error: ErrCodeKeyNotFound},
		},
	}

	if !resp.Success {
		t.Error("expected success")
	}

	if len(resp.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(resp.Results))
	}

	// Verify individual results
	if !resp.Results[0].Success {
		t.Error("expected result 0 to succeed")
	}

	if !resp.Results[1].Success {
		t.Error("expected result 1 to succeed")
	}

	if resp.Results[2].Success {
		t.Error("expected result 2 to fail")
	}

	if resp.Results[2].Error != ErrCodeKeyNotFound {
		t.Errorf("expected error %s, got %s", ErrCodeKeyNotFound, resp.Results[2].Error)
	}
}

func TestConvertToBatchResponse(t *testing.T) {
	resp := &Response{
		Success:       false,
		Error:         ErrCodeNotLeader,
		LeaderID:      "node-2",
		LeaderAddress: "localhost:8002",
	}

	batchResp := convertToBatchResponse(resp)

	if batchResp.Success != resp.Success {
		t.Error("success mismatch")
	}

	if batchResp.Error != resp.Error {
		t.Error("error mismatch")
	}

	if batchResp.LeaderID != resp.LeaderID {
		t.Error("leader ID mismatch")
	}

	if batchResp.LeaderAddress != resp.LeaderAddress {
		t.Error("leader address mismatch")
	}
}

func TestBatchTimeout(t *testing.T) {
	req := &BatchRequest{
		ID: "req-1",
		Operations: []Operation{
			{Type: OperationPut, Key: "key1", Value: []byte("value1")},
		},
		Timeout: 100 * time.Millisecond,
	}

	if req.Timeout != 100*time.Millisecond {
		t.Errorf("expected timeout 100ms, got %v", req.Timeout)
	}
}

func TestBatchCommand_Structure(t *testing.T) {
	// This tests the BatchCommand structure used internally
	// It doesn't need a full integration test, just structural validation

	batchCmd := BatchCommand{
		Commands: make([]interface{}, 3),
	}

	if len(batchCmd.Commands) != 3 {
		t.Errorf("expected 3 commands, got %d", len(batchCmd.Commands))
	}
}
