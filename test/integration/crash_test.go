package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
)

func TestCrashRecovery_Basic(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	// Create initial node
	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		store.Close()
		t.Fatalf("Failed to create WAL: %v", err)
	}

	persistence := raft.NewPersistenceLayer(store, wal)

	// Persist some state
	if err := persistence.PersistMetadata(5, "node-leader"); err != nil {
		t.Fatalf("Failed to persist metadata: %v", err)
	}

	entries := []*raft.LogEntry{
		{Index: 1, Term: 1, Data: []byte("put key1 value1"), Type: raft.EntryNormal},
		{Index: 2, Term: 1, Data: []byte("put key2 value2"), Type: raft.EntryNormal},
		{Index: 3, Term: 2, Data: []byte("put key3 value3"), Type: raft.EntryNormal},
	}

	if err := persistence.PersistLogEntries(entries); err != nil {
		t.Fatalf("Failed to persist log entries: %v", err)
	}

	// Simulate crash - close everything
	store.Close()
	wal.Close()

	// Recovery - reopen storage
	store, err = storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store.Close()

	wal, err = storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	newPersistence := raft.NewPersistenceLayer(store, wal)
	recovery := raft.NewRecoveryManager(newPersistence)

	// Create new node
	node, err := raft.NewNode("node-1", "localhost:5001", []raft.Peer{})
	if err != nil {
		t.Fatalf("Failed to create node: %v", err)
	}

	// Recover state
	if err := recovery.RecoverState(node); err != nil {
		t.Fatalf("Failed to recover state: %v", err)
	}

	// Verify recovered state
	if node.GetCurrentTerm() != 5 {
		t.Errorf("Expected term 5, got %d", node.GetCurrentTerm())
	}

	recoveredLog := node.GetLog()
	if len(recoveredLog) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(recoveredLog))
	}

	for i, entry := range recoveredLog {
		if entry.Index != entries[i].Index {
			t.Errorf("Entry %d: expected index %d, got %d", i, entries[i].Index, entry.Index)
		}
		if entry.Term != entries[i].Term {
			t.Errorf("Entry %d: expected term %d, got %d", i, entries[i].Term, entry.Term)
		}
	}
}

func TestCrashRecovery_MultipleRestarts(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	// Simulate multiple crash-restart cycles
	for cycle := 1; cycle <= 3; cycle++ {
		store, err := storage.NewBoltStore(storePath)
		if err != nil {
			t.Fatalf("Cycle %d: Failed to create store: %v", cycle, err)
		}

		wal, err := storage.NewWAL(walPath)
		if err != nil {
			store.Close()
			t.Fatalf("Cycle %d: Failed to create WAL: %v", cycle, err)
		}

		persistence := raft.NewPersistenceLayer(store, wal)

		// Add more data each cycle
		term := uint64(cycle * 10)
		if err := persistence.PersistMetadata(term, "node-leader"); err != nil {
			t.Fatalf("Cycle %d: Failed to persist metadata: %v", cycle, err)
		}

		entries := []*raft.LogEntry{
			{
				Index: uint64(cycle),
				Term:  term,
				Data:  []byte("data from cycle"),
				Type:  raft.EntryNormal,
			},
		}

		if err := persistence.PersistLogEntries(entries); err != nil {
			t.Fatalf("Cycle %d: Failed to persist entries: %v", cycle, err)
		}

		// Force sync
		if err := persistence.Sync(); err != nil {
			t.Fatalf("Cycle %d: Failed to sync: %v", cycle, err)
		}

		// Simulate crash
		store.Close()
		wal.Close()

		// Verify data survives
		store, err = storage.NewBoltStore(storePath)
		if err != nil {
			t.Fatalf("Cycle %d: Failed to reopen store: %v", cycle, err)
		}

		recoveredTerm, _, err := persistence.LoadMetadata()
		if err != nil {
			store.Close()
			t.Fatalf("Cycle %d: Failed to load metadata: %v", cycle, err)
		}

		if recoveredTerm != term {
			t.Errorf("Cycle %d: Expected term %d, got %d", cycle, term, recoveredTerm)
		}

		store.Close()
	}
}

func TestCrashRecovery_DataIntegrity(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		store.Close()
		t.Fatalf("Failed to create WAL: %v", err)
	}

	persistence := raft.NewPersistenceLayer(store, wal)

	// Write a large number of entries
	numEntries := 1000
	entries := make([]*raft.LogEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		entries[i] = &raft.LogEntry{
			Index: uint64(i + 1),
			Term:  uint64((i / 100) + 1),
			Data:  []byte("test data for entry"),
			Type:  raft.EntryNormal,
		}
	}

	if err := persistence.PersistLogEntries(entries); err != nil {
		t.Fatalf("Failed to persist entries: %v", err)
	}

	// Force sync
	if err := persistence.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Close (simulate crash)
	store.Close()
	wal.Close()

	// Recovery
	store, err = storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store.Close()

	wal, err = storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	newPersistence := raft.NewPersistenceLayer(store, wal)
	recovery := raft.NewRecoveryManager(newPersistence)

	// Verify integrity
	if err := recovery.VerifyIntegrity(); err != nil {
		t.Fatalf("Integrity check failed: %v", err)
	}

	// Load and verify all entries
	loadedEntries, err := newPersistence.LoadLogEntries()
	if err != nil {
		t.Fatalf("Failed to load entries: %v", err)
	}

	if len(loadedEntries) != numEntries {
		t.Fatalf("Expected %d entries, got %d", numEntries, len(loadedEntries))
	}

	for i, entry := range loadedEntries {
		if entry.Index != entries[i].Index {
			t.Errorf("Entry %d: expected index %d, got %d", i, entries[i].Index, entry.Index)
		}
		if entry.Term != entries[i].Term {
			t.Errorf("Entry %d: expected term %d, got %d", i, entries[i].Term, entry.Term)
		}
	}
}

func TestCrashRecovery_PartialWrite(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		store.Close()
		t.Fatalf("Failed to create WAL: %v", err)
	}

	persistence := raft.NewPersistenceLayer(store, wal)

	// Write some entries and sync
	entries := []*raft.LogEntry{
		{Index: 1, Term: 1, Data: []byte("committed entry"), Type: raft.EntryNormal},
	}

	if err := persistence.PersistLogEntries(entries); err != nil {
		t.Fatalf("Failed to persist entries: %v", err)
	}

	if err := persistence.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Write more entries but don't sync (simulate crash during write)
	moreEntries := []*raft.LogEntry{
		{Index: 2, Term: 1, Data: []byte("uncommitted entry"), Type: raft.EntryNormal},
	}

	// Write to store but don't sync
	store.SaveLogEntries(moreEntries)
	// Simulate crash - close without sync
	store.Close()
	wal.Close()

	// Recovery
	store, err = storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store.Close()

	wal, err = storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	newPersistence := raft.NewPersistenceLayer(store, wal)

	// Load entries - should only have committed entry
	loadedEntries, err := newPersistence.LoadLogEntries()
	if err != nil {
		t.Fatalf("Failed to load entries: %v", err)
	}

	// At least the first entry should be there (synced)
	// The second entry may or may not be there depending on BoltDB buffering
	if len(loadedEntries) < 1 {
		t.Fatalf("Expected at least 1 entry, got %d", len(loadedEntries))
	}

	if loadedEntries[0].Index != 1 {
		t.Errorf("Expected first entry index 1, got %d", loadedEntries[0].Index)
	}
}

func TestCrashRecovery_ConcurrentWrites(t *testing.T) {
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

	persistence := raft.NewPersistenceLayer(store, wal)

	// Concurrent writes
	done := make(chan bool)
	numGoroutines := 10
	entriesPerGoroutine := 10

	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			for i := 0; i < entriesPerGoroutine; i++ {
				entry := &raft.LogEntry{
					Index: uint64(id*entriesPerGoroutine + i + 1),
					Term:  1,
					Data:  []byte("concurrent write"),
					Type:  raft.EntryNormal,
				}

				if err := persistence.PersistLogEntry(entry); err != nil {
					t.Errorf("Failed to persist entry: %v", err)
				}
			}
			done <- true
		}(g)
	}

	// Wait for all writes
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Final sync
	if err := persistence.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Verify all entries
	loadedEntries, err := persistence.LoadLogEntries()
	if err != nil {
		t.Fatalf("Failed to load entries: %v", err)
	}

	expectedTotal := numGoroutines * entriesPerGoroutine
	if len(loadedEntries) != expectedTotal {
		t.Errorf("Expected %d entries, got %d", expectedTotal, len(loadedEntries))
	}
}

func TestCrashRecovery_RapidRestarts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rapid restart test in short mode")
	}

	dir := t.TempDir()
	storePath := filepath.Join(dir, "store.db")
	walPath := filepath.Join(dir, "wal")

	numRestarts := 20

	for i := 0; i < numRestarts; i++ {
		store, err := storage.NewBoltStore(storePath)
		if err != nil {
			t.Fatalf("Restart %d: Failed to create store: %v", i, err)
		}

		wal, err := storage.NewWAL(walPath)
		if err != nil {
			store.Close()
			t.Fatalf("Restart %d: Failed to create WAL: %v", i, err)
		}

		persistence := raft.NewPersistenceLayer(store, wal)

		// Quick write and close
		if err := persistence.PersistMetadata(uint64(i), "node-1"); err != nil {
			t.Fatalf("Restart %d: Failed to persist metadata: %v", i, err)
		}

		if err := persistence.Sync(); err != nil {
			t.Fatalf("Restart %d: Failed to sync: %v", i, err)
		}

		store.Close()
		wal.Close()

		// Small delay between restarts
		time.Sleep(10 * time.Millisecond)
	}

	// Final verification
	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		t.Fatalf("Failed to reopen store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	persistence := raft.NewPersistenceLayer(store, wal)
	term, _, err := persistence.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if term != uint64(numRestarts-1) {
		t.Errorf("Expected term %d, got %d", numRestarts-1, term)
	}
}
