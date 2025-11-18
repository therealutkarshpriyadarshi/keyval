package api

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
)

func TestServer_ValidateRequest(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name    string
		req     *Request
		wantErr bool
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
		},
		{
			name: "empty key",
			req: &Request{
				Type: OperationGet,
				Key:  "",
			},
			wantErr: true,
		},
		{
			name: "put without value",
			req: &Request{
				Type:  OperationPut,
				Key:   "key1",
				Value: nil,
			},
			wantErr: true,
		},
		{
			name: "valid get",
			req: &Request{
				Type: OperationGet,
				Key:  "key1",
			},
			wantErr: false,
		},
		{
			name: "valid put",
			req: &Request{
				Type:  OperationPut,
				Key:   "key1",
				Value: []byte("value1"),
			},
			wantErr: false,
		},
		{
			name: "valid delete",
			req: &Request{
				Type: OperationDelete,
				Key:  "key1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.validateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_Deduplication(t *testing.T) {
	// Create a test node
	node, err := raft.NewNode("test", "127.0.0.1:8001", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}
	defer node.Stop()

	server := NewServer(node)

	// First request
	req1 := &Request{
		ID:       "req1",
		ClientID: "client1",
		Sequence: 1,
		Type:     OperationGet,
		Key:      "key1",
	}

	// Cache a response
	resp1 := &Response{
		Success: true,
		Value:   []byte("value1"),
	}
	server.cacheResponse(req1, resp1)

	// Same request should return cached response
	cached := server.checkDuplicate(req1)
	if cached == nil {
		t.Error("Expected cached response")
	}

	if string(cached.Value) != "value1" {
		t.Errorf("Expected value1, got %s", string(cached.Value))
	}

	// Different sequence should not return cached response
	req2 := &Request{
		ID:       "req2",
		ClientID: "client1",
		Sequence: 2,
		Type:     OperationGet,
		Key:      "key2",
	}

	cached = server.checkDuplicate(req2)
	if cached != nil {
		t.Error("Expected no cached response for new sequence")
	}
}

func TestServer_CleanupSessions(t *testing.T) {
	server := &Server{
		sessions: make(map[string]*ClientSession),
	}

	// Add some sessions
	for i := 0; i < 5; i++ {
		clientID := string(rune('a' + i))
		server.sessions[clientID] = &ClientSession{
			lastSequence:    1,
			lastResponse:    &Response{Success: true},
			lastRequestTime: time.Now().Add(-10 * time.Minute),
		}
	}

	// Add one recent session
	server.sessions["recent"] = &ClientSession{
		lastSequence:    1,
		lastResponse:    &Response{Success: true},
		lastRequestTime: time.Now(),
	}

	// Cleanup old sessions
	server.CleanupSessions(5 * time.Minute)

	// Should only have the recent session
	if len(server.sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(server.sessions))
	}

	if _, exists := server.sessions["recent"]; !exists {
		t.Error("Expected recent session to exist")
	}
}

func TestOperationType_String(t *testing.T) {
	tests := []struct {
		op   OperationType
		want string
	}{
		{OperationPut, "PUT"},
		{OperationGet, "GET"},
		{OperationDelete, "DELETE"},
		{OperationType(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.op.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}
