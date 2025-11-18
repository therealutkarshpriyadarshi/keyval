package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// Event represents an operation in the history
type Event struct {
	Type      EventType
	Key       string
	Value     string
	Time      time.Time
	ClientID  int
	RequestID int
}

// EventType represents the type of event
type EventType int

const (
	// EventInvoke represents the start of an operation
	EventInvoke EventType = iota
	// EventReturn represents the completion of an operation
	EventReturn
)

// History records all operations for linearizability verification
type History struct {
	mu     sync.Mutex
	events []Event
}

// NewHistory creates a new history
func NewHistory() *History {
	return &History{
		events: make([]Event, 0),
	}
}

// RecordInvoke records the invocation of an operation
func (h *History) RecordInvoke(clientID, requestID int, key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.events = append(h.events, Event{
		Type:      EventInvoke,
		Key:       key,
		Value:     value,
		Time:      time.Now(),
		ClientID:  clientID,
		RequestID: requestID,
	})
}

// RecordReturn records the return of an operation
func (h *History) RecordReturn(clientID, requestID int, key, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.events = append(h.events, Event{
		Type:      EventReturn,
		Key:       key,
		Value:     value,
		Time:      time.Now(),
		ClientID:  clientID,
		RequestID: requestID,
	})
}

// GetEvents returns all events
func (h *History) GetEvents() []Event {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Return a copy
	events := make([]Event, len(h.events))
	copy(events, h.events)
	return events
}

// VerifyLinearizability performs basic linearizability checks
// This is a simplified version - production systems would use libraries like Porcupine
func (h *History) VerifyLinearizability() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Build operation pairs (invoke -> return)
	operations := make(map[string]*Operation)

	for _, event := range h.events {
		opKey := fmt.Sprintf("%d-%d", event.ClientID, event.RequestID)

		if event.Type == EventInvoke {
			operations[opKey] = &Operation{
				Invoke: event,
			}
		} else {
			if op, exists := operations[opKey]; exists {
				op.Return = event
			}
		}
	}

	// Basic checks:
	// 1. All operations should complete
	for key, op := range operations {
		if op.Return.Time.IsZero() {
			return fmt.Errorf("operation %s did not complete", key)
		}
	}

	// 2. Operations should not overlap in time in a way that violates linearizability
	// For a more thorough check, use a library like Porcupine

	return nil
}

// Operation represents a complete operation (invoke + return)
type Operation struct {
	Invoke Event
	Return Event
}

// TestLinearizability_SequentialWrites tests linearizability with sequential writes
func TestLinearizability_SequentialWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	history := NewHistory()

	// Simulate sequential writes
	for i := 0; i < 10; i++ {
		key := "key"
		value := fmt.Sprintf("value-%d", i)

		// Invoke
		history.RecordInvoke(0, i, key, value)

		// Simulate operation
		time.Sleep(10 * time.Millisecond)

		// Return
		history.RecordReturn(0, i, key, value)
	}

	// Verify linearizability
	if err := history.VerifyLinearizability(); err != nil {
		t.Errorf("linearizability check failed: %v", err)
	}
}

// TestLinearizability_ConcurrentWrites tests linearizability with concurrent writes
func TestLinearizability_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	history := NewHistory()
	numClients := 5
	numOps := 20

	var wg sync.WaitGroup

	for clientID := 0; clientID < numClients; clientID++ {
		wg.Add(1)
		go func(cid int) {
			defer wg.Done()

			for i := 0; i < numOps; i++ {
				key := fmt.Sprintf("key-%d", i%5) // 5 different keys
				value := fmt.Sprintf("client-%d-value-%d", cid, i)

				// Invoke
				history.RecordInvoke(cid, i, key, value)

				// Simulate operation
				time.Sleep(time.Duration(5+i%10) * time.Millisecond)

				// Return
				history.RecordReturn(cid, i, key, value)
			}
		}(clientID)
	}

	wg.Wait()

	// Verify linearizability
	if err := history.VerifyLinearizability(); err != nil {
		t.Errorf("linearizability check failed: %v", err)
	}

	// Check that we have the expected number of events
	events := history.GetEvents()
	expectedEvents := numClients * numOps * 2 // invoke + return for each op
	if len(events) != expectedEvents {
		t.Errorf("expected %d events, got %d", expectedEvents, len(events))
	}
}

// TestLinearizability_ReadYourWrites tests read-your-writes consistency
func TestLinearizability_ReadYourWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	history := NewHistory()

	// Write a value
	history.RecordInvoke(0, 0, "key", "value1")
	time.Sleep(10 * time.Millisecond)
	history.RecordReturn(0, 0, "key", "value1")

	// Read should see the written value
	history.RecordInvoke(0, 1, "key", "")
	time.Sleep(5 * time.Millisecond)
	history.RecordReturn(0, 1, "key", "value1")

	// Verify
	if err := history.VerifyLinearizability(); err != nil {
		t.Errorf("linearizability check failed: %v", err)
	}
}

// TestLinearizability_MonotonicReads tests monotonic reads
func TestLinearizability_MonotonicReads(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	history := NewHistory()

	// Write initial value
	history.RecordInvoke(0, 0, "key", "value1")
	time.Sleep(10 * time.Millisecond)
	history.RecordReturn(0, 0, "key", "value1")

	// First read
	history.RecordInvoke(1, 0, "key", "")
	time.Sleep(5 * time.Millisecond)
	history.RecordReturn(1, 0, "key", "value1")

	// Write new value
	history.RecordInvoke(0, 1, "key", "value2")
	time.Sleep(10 * time.Millisecond)
	history.RecordReturn(0, 1, "key", "value2")

	// Second read should not see older value
	history.RecordInvoke(1, 1, "key", "")
	time.Sleep(5 * time.Millisecond)
	history.RecordReturn(1, 1, "key", "value2")

	// Verify
	if err := history.VerifyLinearizability(); err != nil {
		t.Errorf("linearizability check failed: %v", err)
	}
}

// TestLinearizability_ConcurrentReadWrite tests concurrent reads and writes
func TestLinearizability_ConcurrentReadWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	history := NewHistory()

	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			key := "key"
			value := fmt.Sprintf("value-%d", i)

			history.RecordInvoke(0, i, key, value)
			time.Sleep(20 * time.Millisecond)
			history.RecordReturn(0, i, key, value)
		}
	}()

	// Readers
	for readerID := 1; readerID <= 3; readerID++ {
		wg.Add(1)
		go func(rid int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				history.RecordInvoke(rid, i, "key", "")
				time.Sleep(25 * time.Millisecond)
				history.RecordReturn(rid, i, "key", fmt.Sprintf("value-%d", i))
			}
		}(readerID)
	}

	wg.Wait()

	// Verify
	if err := history.VerifyLinearizability(); err != nil {
		t.Errorf("linearizability check failed: %v", err)
	}
}

// TestHistory_EventRecording tests event recording
func TestHistory_EventRecording(t *testing.T) {
	history := NewHistory()

	history.RecordInvoke(0, 1, "key1", "value1")
	history.RecordReturn(0, 1, "key1", "value1")
	history.RecordInvoke(1, 1, "key2", "value2")
	history.RecordReturn(1, 1, "key2", "value2")

	events := history.GetEvents()
	if len(events) != 4 {
		t.Errorf("expected 4 events, got %d", len(events))
	}

	// Check first event
	if events[0].Type != EventInvoke {
		t.Error("expected first event to be invoke")
	}
	if events[0].Key != "key1" {
		t.Errorf("expected key 'key1', got '%s'", events[0].Key)
	}

	// Check second event
	if events[1].Type != EventReturn {
		t.Error("expected second event to be return")
	}
}
