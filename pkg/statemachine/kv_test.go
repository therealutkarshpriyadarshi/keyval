package statemachine

import (
	"encoding/json"
	"testing"
)

func TestKVStateMachine_Put(t *testing.T) {
	kv := NewKVStateMachine()

	// Create a Put command
	cmd := Command{
		Type:  CommandPut,
		Key:   "key1",
		Value: []byte("value1"),
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("Failed to marshal command: %v", err)
	}

	// Apply the command
	result, err := kv.Apply(data)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	resp, ok := result.(*CommandResponse)
	if !ok {
		t.Fatalf("Expected CommandResponse, got %T", result)
	}

	if !resp.Success {
		t.Errorf("Expected success, got failure")
	}

	// Verify the value was stored
	value, exists := kv.Get("key1")
	if !exists {
		t.Errorf("Expected key1 to exist")
	}

	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", string(value))
	}
}

func TestKVStateMachine_Delete(t *testing.T) {
	kv := NewKVStateMachine()

	// First, put a value
	putCmd := Command{
		Type:  CommandPut,
		Key:   "key1",
		Value: []byte("value1"),
	}

	data, _ := json.Marshal(putCmd)
	kv.Apply(data)

	// Verify it exists
	_, exists := kv.Get("key1")
	if !exists {
		t.Fatal("Key should exist before delete")
	}

	// Now delete it
	delCmd := Command{
		Type: CommandDelete,
		Key:  "key1",
	}

	data, _ = json.Marshal(delCmd)
	result, err := kv.Apply(data)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	resp, ok := result.(*CommandResponse)
	if !ok {
		t.Fatalf("Expected CommandResponse, got %T", result)
	}

	if !resp.Success {
		t.Errorf("Expected success, got failure")
	}

	// Verify it's deleted
	_, exists = kv.Get("key1")
	if exists {
		t.Errorf("Key should not exist after delete")
	}
}

func TestKVStateMachine_Get(t *testing.T) {
	kv := NewKVStateMachine()

	// Put a value
	putCmd := Command{
		Type:  CommandPut,
		Key:   "key1",
		Value: []byte("value1"),
	}

	data, _ := json.Marshal(putCmd)
	kv.Apply(data)

	// Get the value directly
	value, exists := kv.Get("key1")
	if !exists {
		t.Fatal("Key should exist")
	}

	if string(value) != "value1" {
		t.Errorf("Expected value1, got %s", string(value))
	}

	// Get non-existent key
	_, exists = kv.Get("nonexistent")
	if exists {
		t.Error("Non-existent key should not exist")
	}
}

func TestKVStateMachine_Snapshot(t *testing.T) {
	kv := NewKVStateMachine()

	// Add some data
	for i := 0; i < 10; i++ {
		cmd := Command{
			Type:  CommandPut,
			Key:   string(rune('a' + i)),
			Value: []byte{byte(i)},
		}
		data, _ := json.Marshal(cmd)
		kv.Apply(data)
	}

	// Create a snapshot
	snapshot, err := kv.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	if len(snapshot) == 0 {
		t.Error("Snapshot should not be empty")
	}

	// Create a new state machine and restore
	kv2 := NewKVStateMachine()
	if err := kv2.Restore(snapshot); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify all data was restored
	for i := 0; i < 10; i++ {
		key := string(rune('a' + i))
		value, exists := kv2.Get(key)
		if !exists {
			t.Errorf("Key %s should exist after restore", key)
		}
		if len(value) != 1 || value[0] != byte(i) {
			t.Errorf("Value mismatch for key %s", key)
		}
	}
}

func TestKVStateMachine_Size(t *testing.T) {
	kv := NewKVStateMachine()

	if kv.Size() != 0 {
		t.Error("Empty state machine should have size 0")
	}

	// Add some entries
	for i := 0; i < 5; i++ {
		cmd := Command{
			Type:  CommandPut,
			Key:   string(rune('a' + i)),
			Value: []byte{byte(i)},
		}
		data, _ := json.Marshal(cmd)
		kv.Apply(data)
	}

	if kv.Size() != 5 {
		t.Errorf("Expected size 5, got %d", kv.Size())
	}
}

func TestKVStateMachine_Keys(t *testing.T) {
	kv := NewKVStateMachine()

	// Add some entries
	expectedKeys := []string{"key1", "key2", "key3"}
	for _, key := range expectedKeys {
		cmd := Command{
			Type:  CommandPut,
			Key:   key,
			Value: []byte("value"),
		}
		data, _ := json.Marshal(cmd)
		kv.Apply(data)
	}

	keys := kv.Keys()
	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	for _, expected := range expectedKeys {
		if !keyMap[expected] {
			t.Errorf("Expected key %s not found", expected)
		}
	}
}

func TestKVStateMachine_Clear(t *testing.T) {
	kv := NewKVStateMachine()

	// Add some entries
	for i := 0; i < 5; i++ {
		cmd := Command{
			Type:  CommandPut,
			Key:   string(rune('a' + i)),
			Value: []byte{byte(i)},
		}
		data, _ := json.Marshal(cmd)
		kv.Apply(data)
	}

	if kv.Size() != 5 {
		t.Fatal("Expected size 5 before clear")
	}

	// Clear the state machine
	kv.Clear()

	if kv.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", kv.Size())
	}
}

func TestKVStateMachine_Concurrency(t *testing.T) {
	kv := NewKVStateMachine()

	// Test concurrent puts
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			cmd := Command{
				Type:  CommandPut,
				Key:   string(rune('a' + idx)),
				Value: []byte{byte(idx)},
			}
			data, _ := json.Marshal(cmd)
			kv.Apply(data)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if kv.Size() != 10 {
		t.Errorf("Expected size 10, got %d", kv.Size())
	}
}
