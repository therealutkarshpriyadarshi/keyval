package raft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
)

func TestRecoveryManager_RecoverState(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	// Create storage
	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Create a node
	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Set some initial state
	node.serverState.SetCurrentTerm(5)
	node.serverState.SetVotedFor("node-2")
	node.serverState.mu.Lock()
	node.serverState.Log = []LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Index: 2, Term: 2, Data: []byte("cmd2"), Type: EntryNormal},
	}
	node.serverState.mu.Unlock()

	// Persist the state
	if err := persistence.PersistMetadata(5, "node-2"); err != nil {
		t.Fatalf("Failed to persist metadata: %v", err)
	}

	entries := node.serverState.GetLog()
	logEntries := make([]*LogEntry, len(entries))
	for i := range entries {
		logEntries[i] = &entries[i]
	}
	if err := persistence.PersistLogEntries(logEntries); err != nil {
		t.Fatalf("Failed to persist log entries: %v", err)
	}

	// Create a new node (simulating restart)
	newNode, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create new node: %v", err)
	}

	// Recover state
	if err := recovery.RecoverState(newNode); err != nil {
		t.Fatalf("Failed to recover state: %v", err)
	}

	// Verify recovered state
	if newNode.serverState.GetCurrentTerm() != 5 {
		t.Errorf("Expected term 5, got %d", newNode.serverState.GetCurrentTerm())
	}

	if newNode.serverState.GetVotedFor() != "node-2" {
		t.Errorf("Expected votedFor node-2, got %s", newNode.serverState.GetVotedFor())
	}

	recoveredLog := newNode.serverState.GetLog()
	if len(recoveredLog) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(recoveredLog))
	}
}

func TestRecoveryManager_ValidateSnapshot(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Valid snapshot
	validSnapshot := &storage.SnapshotMetadata{
		LastIncludedIndex: 100,
		LastIncludedTerm:  5,
		Size:              1024,
	}

	if err := recovery.validateSnapshot(validSnapshot); err != nil {
		t.Errorf("Valid snapshot should pass validation: %v", err)
	}

	// Invalid snapshot - zero index
	invalidSnapshot1 := &storage.SnapshotMetadata{
		LastIncludedIndex: 0,
		LastIncludedTerm:  5,
		Size:              1024,
	}

	if err := recovery.validateSnapshot(invalidSnapshot1); err == nil {
		t.Error("Snapshot with zero index should fail validation")
	}

	// Invalid snapshot - zero size
	invalidSnapshot2 := &storage.SnapshotMetadata{
		LastIncludedIndex: 100,
		LastIncludedTerm:  5,
		Size:              0,
	}

	if err := recovery.validateSnapshot(invalidSnapshot2); err == nil {
		t.Error("Snapshot with zero size should fail validation")
	}
}

func TestRecoveryManager_ValidateRecoveredState(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Valid state
	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	node.serverState.mu.Lock()
	node.serverState.Log = []LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Index: 2, Term: 2, Data: []byte("cmd2"), Type: EntryNormal},
		{Index: 3, Term: 2, Data: []byte("cmd3"), Type: EntryNormal},
	}
	node.serverState.mu.Unlock()

	if err := recovery.validateRecoveredState(node); err != nil {
		t.Errorf("Valid state should pass validation: %v", err)
	}

	// Invalid state - non-sequential indices
	invalidNode, err := NewNode("node-2", "localhost:5002", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	invalidNode.serverState.mu.Lock()
	invalidNode.serverState.Log = []LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Index: 3, Term: 2, Data: []byte("cmd3"), Type: EntryNormal}, // Skip index 2
	}
	invalidNode.serverState.mu.Unlock()

	if err := recovery.validateRecoveredState(invalidNode); err == nil {
		t.Error("State with non-sequential indices should fail validation")
	}
}

func TestRecoveryManager_CreateCheckpoint(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Create a node with state
	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	node.serverState.SetCurrentTerm(10)
	node.serverState.SetVotedFor("node-3")
	node.serverState.mu.Lock()
	node.serverState.Log = []LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Index: 2, Term: 2, Data: []byte("cmd2"), Type: EntryNormal},
	}
	node.serverState.mu.Unlock()

	// Create checkpoint
	if err := recovery.CreateCheckpoint(node); err != nil {
		t.Fatalf("Failed to create checkpoint: %v", err)
	}

	// Verify checkpoint was saved
	loadedTerm, loadedVotedFor, err := persistence.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedTerm != 10 {
		t.Errorf("Expected term 10, got %d", loadedTerm)
	}

	if loadedVotedFor != "node-3" {
		t.Errorf("Expected votedFor node-3, got %s", loadedVotedFor)
	}

	loadedEntries, err := persistence.LoadLogEntries()
	if err != nil {
		t.Fatalf("Failed to load log entries: %v", err)
	}

	if len(loadedEntries) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(loadedEntries))
	}
}

func TestRecoveryManager_VerifyIntegrity(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Save valid data
	if err := persistence.PersistMetadata(1, "node-1"); err != nil {
		t.Fatalf("Failed to persist metadata: %v", err)
	}

	entries := []*LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Index: 2, Term: 1, Data: []byte("cmd2"), Type: EntryNormal},
	}
	if err := persistence.PersistLogEntries(entries); err != nil {
		t.Fatalf("Failed to persist log entries: %v", err)
	}

	// Verify integrity
	if err := recovery.VerifyIntegrity(); err != nil {
		t.Errorf("Integrity verification should pass: %v", err)
	}
}

func TestRecoveryManager_RecoverFromWAL(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Write some WAL entries
	walEntries := []*storage.WALEntry{
		{Type: storage.WALEntryMetadata, Index: 0, Term: 1, Data: []byte("node-1")},
		{Type: storage.WALEntryLog, Index: 1, Term: 1, Data: []byte("cmd1")},
		{Type: storage.WALEntryLog, Index: 2, Term: 1, Data: []byte("cmd2")},
	}

	for _, entry := range walEntries {
		entry.Checksum = entry.CalculateChecksum()
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append WAL entry: %v", err)
		}
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync WAL: %v", err)
	}

	// Recover from WAL
	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	if err := recovery.RecoverFromWAL(node); err != nil {
		t.Errorf("Failed to recover from WAL: %v", err)
	}
}

func TestRecoveryManager_EmptyLog(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	// Create a node with empty log
	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Recover state (should succeed with empty log)
	if err := recovery.RecoverState(node); err != nil {
		t.Errorf("Recovery should succeed with empty log: %v", err)
	}

	// Verify state
	if node.serverState.GetCurrentTerm() != 0 {
		t.Errorf("Expected term 0, got %d", node.serverState.GetCurrentTerm())
	}

	if node.serverState.GetVotedFor() != "" {
		t.Errorf("Expected empty votedFor, got %s", node.serverState.GetVotedFor())
	}

	log := node.serverState.GetLog()
	if len(log) != 0 {
		t.Errorf("Expected empty log, got %d entries", len(log))
	}
}

func TestRecoveryManager_MultipleCheckpoints(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := NewPersistenceLayer(store, wal)
	recovery := NewRecoveryManager(persistence)

	node, err := NewNode("node-1", "localhost:5001", []Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Create multiple checkpoints with different state
	for i := uint64(1); i <= 5; i++ {
		node.serverState.SetCurrentTerm(i)
		node.serverState.mu.Lock()
		node.serverState.Log = append(node.serverState.Log, LogEntry{
			Index: i,
			Term:  i,
			Data:  []byte("cmd"),
			Type:  EntryNormal,
		})
		node.serverState.mu.Unlock()

		if err := recovery.CreateCheckpoint(node); err != nil {
			t.Fatalf("Failed to create checkpoint %d: %v", i, err)
		}
	}

	// Verify final state
	loadedTerm, _, err := persistence.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedTerm != 5 {
		t.Errorf("Expected term 5, got %d", loadedTerm)
	}

	loadedEntries, err := persistence.LoadLogEntries()
	if err != nil {
		t.Fatalf("Failed to load log entries: %v", err)
	}

	if len(loadedEntries) != 5 {
		t.Errorf("Expected 5 log entries, got %d", len(loadedEntries))
	}
}
