package api

import (
	"testing"
	"time"
)

func TestSessionManager_GetOrCreateSession(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	clientID := "client-1"

	// Create session
	session := sm.GetOrCreateSession(clientID)
	if session == nil {
		t.Fatal("expected session to be created")
	}

	if session.clientID != clientID {
		t.Errorf("expected client ID %s, got %s", clientID, session.clientID)
	}

	// Get same session
	session2 := sm.GetOrCreateSession(clientID)
	if session != session2 {
		t.Error("expected to get the same session instance")
	}

	// Verify session count
	if count := sm.SessionCount(); count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}
}

func TestSessionManager_CheckDuplicate(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	clientID := "client-1"

	// No duplicate for non-existent session
	resp, isDup := sm.CheckDuplicate(clientID, 1)
	if isDup {
		t.Error("expected no duplicate for non-existent session")
	}
	if resp != nil {
		t.Error("expected nil response")
	}

	// Cache a response
	cachedResp := &Response{
		Success: true,
		Value:   []byte("test"),
	}
	sm.CacheResponse(clientID, 1, cachedResp)

	// Check for duplicate with same sequence
	resp, isDup = sm.CheckDuplicate(clientID, 1)
	if !isDup {
		t.Error("expected duplicate for same sequence")
	}
	if resp != cachedResp {
		t.Error("expected cached response to be returned")
	}

	// Check for duplicate with older sequence
	resp, isDup = sm.CheckDuplicate(clientID, 0)
	if !isDup {
		t.Error("expected duplicate for older sequence")
	}

	// No duplicate for newer sequence
	resp, isDup = sm.CheckDuplicate(clientID, 2)
	if isDup {
		t.Error("expected no duplicate for newer sequence")
	}
}

func TestSessionManager_CacheResponse(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	clientID := "client-1"

	// Cache responses with increasing sequence numbers
	resp1 := &Response{Success: true, Value: []byte("v1")}
	resp2 := &Response{Success: true, Value: []byte("v2")}
	resp3 := &Response{Success: true, Value: []byte("v3")}

	sm.CacheResponse(clientID, 1, resp1)
	sm.CacheResponse(clientID, 2, resp2)
	sm.CacheResponse(clientID, 3, resp3)

	// Verify last response is cached
	session, exists := sm.GetSession(clientID)
	if !exists {
		t.Fatal("expected session to exist")
	}

	session.mu.RLock()
	if session.lastSequence != 3 {
		t.Errorf("expected last sequence 3, got %d", session.lastSequence)
	}
	if session.lastResponse != resp3 {
		t.Error("expected last response to be resp3")
	}
	session.mu.RUnlock()

	// Try to cache older response (should not update)
	oldResp := &Response{Success: false}
	sm.CacheResponse(clientID, 1, oldResp)

	session.mu.RLock()
	if session.lastSequence != 3 {
		t.Error("expected sequence to remain 3")
	}
	if session.lastResponse != resp3 {
		t.Error("expected response to remain resp3")
	}
	session.mu.RUnlock()
}

func TestSessionManager_Cleanup(t *testing.T) {
	// Use short max age for testing
	maxAge := 100 * time.Millisecond
	sm := NewSessionManager(maxAge)
	defer sm.Stop()

	// Create some sessions
	for i := 0; i < 5; i++ {
		clientID := string(rune('A' + i))
		sm.CacheResponse(clientID, 1, &Response{Success: true})
	}

	if count := sm.SessionCount(); count != 5 {
		t.Errorf("expected 5 sessions, got %d", count)
	}

	// Wait for sessions to age
	time.Sleep(maxAge + 50*time.Millisecond)

	// Trigger cleanup manually
	sm.cleanup()

	// All sessions should be removed
	if count := sm.SessionCount(); count != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", count)
	}
}

func TestSessionManager_GetSessionInfo(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	clientID := "client-1"

	// No info for non-existent session
	info := sm.GetSessionInfo(clientID)
	if info != nil {
		t.Error("expected nil info for non-existent session")
	}

	// Create session and get info
	sm.CacheResponse(clientID, 5, &Response{Success: true})

	info = sm.GetSessionInfo(clientID)
	if info == nil {
		t.Fatal("expected session info")
	}

	if info.ClientID != clientID {
		t.Errorf("expected client ID %s, got %s", clientID, info.ClientID)
	}

	if info.LastSequence != 5 {
		t.Errorf("expected last sequence 5, got %d", info.LastSequence)
	}

	if info.Age == 0 {
		t.Error("expected non-zero age")
	}
}

func TestSessionManager_ConcurrentAccess(t *testing.T) {
	sm := NewSessionManager(5 * time.Minute)
	defer sm.Stop()

	// Concurrent access from multiple goroutines
	numClients := 10
	numRequests := 100

	done := make(chan struct{})

	for i := 0; i < numClients; i++ {
		go func(clientNum int) {
			clientID := string(rune('A' + clientNum))

			for seq := uint64(1); seq <= uint64(numRequests); seq++ {
				// Cache response
				resp := &Response{
					Success: true,
					Value:   []byte{byte(seq)},
				}
				sm.CacheResponse(clientID, seq, resp)

				// Check for duplicates
				sm.CheckDuplicate(clientID, seq-1)

				// Get session info
				sm.GetSessionInfo(clientID)
			}

			done <- struct{}{}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numClients; i++ {
		<-done
	}

	// Verify session count
	if count := sm.SessionCount(); count != numClients {
		t.Errorf("expected %d sessions, got %d", numClients, count)
	}

	// Verify each session has correct last sequence
	for i := 0; i < numClients; i++ {
		clientID := string(rune('A' + i))
		info := sm.GetSessionInfo(clientID)

		if info == nil {
			t.Errorf("expected session info for client %s", clientID)
			continue
		}

		if info.LastSequence != uint64(numRequests) {
			t.Errorf("client %s: expected last sequence %d, got %d",
				clientID, numRequests, info.LastSequence)
		}
	}
}
