package api

import (
	"sync"
	"time"
)

// SessionManager manages client sessions for request deduplication
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ClientSession

	// Configuration
	maxSessionAge time.Duration
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// ClientSession tracks client request history for deduplication
type ClientSession struct {
	mu              sync.RWMutex
	clientID        string
	lastSequence    uint64
	lastResponse    *Response
	lastRequestTime time.Time
	createdAt       time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager(maxSessionAge time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:      make(map[string]*ClientSession),
		maxSessionAge: maxSessionAge,
		stopCleanup:   make(chan struct{}),
	}

	// Start background cleanup
	sm.startCleanup()

	return sm
}

// GetOrCreateSession gets an existing session or creates a new one
func (sm *SessionManager) GetOrCreateSession(clientID string) *ClientSession {
	sm.mu.RLock()
	session, exists := sm.sessions[clientID]
	sm.mu.RUnlock()

	if exists {
		return session
	}

	// Create new session
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock
	if session, exists := sm.sessions[clientID]; exists {
		return session
	}

	session = &ClientSession{
		clientID:        clientID,
		lastSequence:    0,
		createdAt:       time.Now(),
		lastRequestTime: time.Now(),
	}
	sm.sessions[clientID] = session

	return session
}

// GetSession retrieves an existing session
func (sm *SessionManager) GetSession(clientID string) (*ClientSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[clientID]
	return session, exists
}

// CheckDuplicate checks if a request is a duplicate
func (sm *SessionManager) CheckDuplicate(clientID string, sequence uint64) (*Response, bool) {
	session, exists := sm.GetSession(clientID)
	if !exists {
		return nil, false
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// If we've seen this sequence number, return cached response
	if sequence <= session.lastSequence {
		return session.lastResponse, true
	}

	return nil, false
}

// CacheResponse caches a response for deduplication
func (sm *SessionManager) CacheResponse(clientID string, sequence uint64, resp *Response) {
	session := sm.GetOrCreateSession(clientID)

	session.mu.Lock()
	defer session.mu.Unlock()

	// Only update if this is a newer sequence
	if sequence > session.lastSequence {
		session.lastSequence = sequence
		session.lastResponse = resp
		session.lastRequestTime = time.Now()
	}
}

// startCleanup starts background session cleanup
func (sm *SessionManager) startCleanup() {
	// Cleanup every 1 minute
	sm.cleanupTicker = time.NewTicker(1 * time.Minute)

	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.cleanup()
			case <-sm.stopCleanup:
				return
			}
		}
	}()
}

// cleanup removes old sessions
func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for clientID, session := range sm.sessions {
		session.mu.RLock()
		age := now.Sub(session.lastRequestTime)
		session.mu.RUnlock()

		if age > sm.maxSessionAge {
			delete(sm.sessions, clientID)
		}
	}
}

// Stop stops the session manager and cleanup goroutine
func (sm *SessionManager) Stop() {
	if sm.cleanupTicker != nil {
		sm.cleanupTicker.Stop()
	}
	close(sm.stopCleanup)
}

// SessionCount returns the number of active sessions
func (sm *SessionManager) SessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GetSessionInfo returns information about a session
func (sm *SessionManager) GetSessionInfo(clientID string) *SessionInfo {
	session, exists := sm.GetSession(clientID)
	if !exists {
		return nil
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	return &SessionInfo{
		ClientID:        session.clientID,
		LastSequence:    session.lastSequence,
		LastRequestTime: session.lastRequestTime,
		CreatedAt:       session.createdAt,
		Age:             time.Since(session.createdAt),
	}
}

// SessionInfo contains information about a client session
type SessionInfo struct {
	ClientID        string
	LastSequence    uint64
	LastRequestTime time.Time
	CreatedAt       time.Time
	Age             time.Duration
}
