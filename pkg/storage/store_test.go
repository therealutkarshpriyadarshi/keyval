package storage

import (
	"path/filepath"
	"testing"
)

func TestBoltStore_NewBoltStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("Database should be initialized")
	}
}

func TestBoltStore_SaveAndLoadMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save metadata
	meta := &Metadata{
		CurrentTerm: 5,
		VotedFor:    "node-1",
	}

	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Load metadata
	loadedMeta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMeta.CurrentTerm != meta.CurrentTerm {
		t.Errorf("Expected term %d, got %d", meta.CurrentTerm, loadedMeta.CurrentTerm)
	}

	if loadedMeta.VotedFor != meta.VotedFor {
		t.Errorf("Expected votedFor %s, got %s", meta.VotedFor, loadedMeta.VotedFor)
	}
}

func TestBoltStore_SaveAndLoadLogEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save log entry
	entry := &LogEntry{
		Index: 1,
		Term:  1,
		Data:  []byte("test command"),
		Type:  0, // EntryNormal
	}

	if err := store.SaveLogEntry(entry); err != nil {
		t.Fatalf("Failed to save log entry: %v", err)
	}

	// Load log entry
	loadedEntry, err := store.LoadLogEntry(1)
	if err != nil {
		t.Fatalf("Failed to load log entry: %v", err)
	}

	if loadedEntry.Index != entry.Index {
		t.Errorf("Expected index %d, got %d", entry.Index, loadedEntry.Index)
	}

	if loadedEntry.Term != entry.Term {
		t.Errorf("Expected term %d, got %d", entry.Term, loadedEntry.Term)
	}

	if string(loadedEntry.Data) != string(entry.Data) {
		t.Errorf("Expected data %s, got %s", entry.Data, loadedEntry.Data)
	}
}

func TestBoltStore_SaveAndLoadLogEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save multiple entries
	entries := []*LogEntry{
		{Index: 1, Term: 1, Data: []byte("command 1"), Type: 0},
		{Index: 2, Term: 1, Data: []byte("command 2"), Type: 0},
		{Index: 3, Term: 2, Data: []byte("command 3"), Type: 0},
	}

	if err := store.SaveLogEntries(entries); err != nil {
		t.Fatalf("Failed to save log entries: %v", err)
	}

	// Load all entries
	loadedEntries, err := store.LoadAllLogEntries()
	if err != nil {
		t.Fatalf("Failed to load log entries: %v", err)
	}

	if len(loadedEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(loadedEntries))
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

func TestBoltStore_LoadLogEntriesRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save entries
	entries := []*LogEntry{
		{Index: 1, Term: 1, Data: []byte("command 1"), Type: 0},
		{Index: 2, Term: 1, Data: []byte("command 2"), Type: 0},
		{Index: 3, Term: 2, Data: []byte("command 3"), Type: 0},
		{Index: 4, Term: 2, Data: []byte("command 4"), Type: 0},
		{Index: 5, Term: 3, Data: []byte("command 5"), Type: 0},
	}

	if err := store.SaveLogEntries(entries); err != nil {
		t.Fatalf("Failed to save log entries: %v", err)
	}

	// Load range [2, 4]
	loadedEntries, err := store.LoadLogEntries(2, 4)
	if err != nil {
		t.Fatalf("Failed to load log entries range: %v", err)
	}

	if len(loadedEntries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(loadedEntries))
	}

	if loadedEntries[0].Index != 2 {
		t.Errorf("Expected first entry index 2, got %d", loadedEntries[0].Index)
	}

	if loadedEntries[len(loadedEntries)-1].Index != 4 {
		t.Errorf("Expected last entry index 4, got %d", loadedEntries[len(loadedEntries)-1].Index)
	}
}

func TestBoltStore_DeleteLogEntriesFrom(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save entries
	entries := []*LogEntry{
		{Index: 1, Term: 1, Data: []byte("command 1"), Type: 0},
		{Index: 2, Term: 1, Data: []byte("command 2"), Type: 0},
		{Index: 3, Term: 2, Data: []byte("command 3"), Type: 0},
		{Index: 4, Term: 2, Data: []byte("command 4"), Type: 0},
	}

	if err := store.SaveLogEntries(entries); err != nil {
		t.Fatalf("Failed to save log entries: %v", err)
	}

	// Delete entries from index 3
	if err := store.DeleteLogEntriesFrom(3); err != nil {
		t.Fatalf("Failed to delete log entries: %v", err)
	}

	// Load all entries
	loadedEntries, err := store.LoadAllLogEntries()
	if err != nil {
		t.Fatalf("Failed to load log entries: %v", err)
	}

	if len(loadedEntries) != 2 {
		t.Fatalf("Expected 2 entries after delete, got %d", len(loadedEntries))
	}

	if loadedEntries[len(loadedEntries)-1].Index != 2 {
		t.Errorf("Expected last entry index 2, got %d", loadedEntries[len(loadedEntries)-1].Index)
	}
}

func TestBoltStore_FirstAndLastIndex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Empty store
	firstIndex, err := store.FirstIndex()
	if err != nil {
		t.Fatalf("Failed to get first index: %v", err)
	}
	if firstIndex != 0 {
		t.Errorf("Expected first index 0 for empty store, got %d", firstIndex)
	}

	lastIndex, err := store.LastIndex()
	if err != nil {
		t.Fatalf("Failed to get last index: %v", err)
	}
	if lastIndex != 0 {
		t.Errorf("Expected last index 0 for empty store, got %d", lastIndex)
	}

	// Save entries
	entries := []*LogEntry{
		{Index: 5, Term: 1, Data: []byte("command 5"), Type: 0},
		{Index: 6, Term: 1, Data: []byte("command 6"), Type: 0},
		{Index: 7, Term: 2, Data: []byte("command 7"), Type: 0},
	}

	if err := store.SaveLogEntries(entries); err != nil {
		t.Fatalf("Failed to save log entries: %v", err)
	}

	// Get indices
	firstIndex, err = store.FirstIndex()
	if err != nil {
		t.Fatalf("Failed to get first index: %v", err)
	}
	if firstIndex != 5 {
		t.Errorf("Expected first index 5, got %d", firstIndex)
	}

	lastIndex, err = store.LastIndex()
	if err != nil {
		t.Fatalf("Failed to get last index: %v", err)
	}
	if lastIndex != 7 {
		t.Errorf("Expected last index 7, got %d", lastIndex)
	}
}

func TestBoltStore_SaveAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save snapshot metadata
	meta := &SnapshotMetadata{
		LastIncludedIndex: 100,
		LastIncludedTerm:  5,
		Size:              1024,
		Checksum:          12345,
	}

	if err := store.SaveSnapshot(meta); err != nil {
		t.Fatalf("Failed to save snapshot: %v", err)
	}

	// Load snapshot metadata
	loadedMeta, err := store.LoadSnapshot()
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	if loadedMeta.LastIncludedIndex != meta.LastIncludedIndex {
		t.Errorf("Expected lastIncludedIndex %d, got %d",
			meta.LastIncludedIndex, loadedMeta.LastIncludedIndex)
	}

	if loadedMeta.LastIncludedTerm != meta.LastIncludedTerm {
		t.Errorf("Expected lastIncludedTerm %d, got %d",
			meta.LastIncludedTerm, loadedMeta.LastIncludedTerm)
	}
}

func TestBoltStore_Persistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	// Create store and save data
	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}

	meta := &Metadata{CurrentTerm: 10, VotedFor: "node-2"}
	if err := store.SaveMetadata(meta); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	entries := []*LogEntry{
		{Index: 1, Term: 1, Data: []byte("cmd1"), Type: 0},
		{Index: 2, Term: 1, Data: []byte("cmd2"), Type: 0},
	}
	if err := store.SaveLogEntries(entries); err != nil {
		t.Fatalf("Failed to save entries: %v", err)
	}

	store.Close()

	// Reopen store and verify data
	store, err = NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to reopen BoltStore: %v", err)
	}
	defer store.Close()

	loadedMeta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMeta.CurrentTerm != meta.CurrentTerm {
		t.Errorf("Expected term %d, got %d", meta.CurrentTerm, loadedMeta.CurrentTerm)
	}

	loadedEntries, err := store.LoadAllLogEntries()
	if err != nil {
		t.Fatalf("Failed to load entries: %v", err)
	}

	if len(loadedEntries) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(loadedEntries))
	}
}

func TestBoltStore_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Concurrent writes
	done := make(chan bool)
	for i := 1; i <= 10; i++ {
		go func(index int) {
			entry := &LogEntry{
				Index: uint64(index),
				Term:  1,
				Data:  []byte("concurrent"),
				Type:  0, // EntryNormal
			}

			if err := store.SaveLogEntry(entry); err != nil {
				t.Errorf("Failed to save entry %d: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all entries
	entries, err := store.LoadAllLogEntries()
	if err != nil {
		t.Fatalf("Failed to load entries: %v", err)
	}

	if len(entries) != 10 {
		t.Errorf("Expected 10 entries, got %d", len(entries))
	}
}

func TestBoltStore_UpdateMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	store, err := NewBoltStore(path)
	if err != nil {
		t.Fatalf("Failed to create BoltStore: %v", err)
	}
	defer store.Close()

	// Save initial metadata
	meta1 := &Metadata{CurrentTerm: 1, VotedFor: "node-1"}
	if err := store.SaveMetadata(meta1); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	// Update metadata
	meta2 := &Metadata{CurrentTerm: 2, VotedFor: "node-2"}
	if err := store.SaveMetadata(meta2); err != nil {
		t.Fatalf("Failed to update metadata: %v", err)
	}

	// Load and verify
	loadedMeta, err := store.LoadMetadata()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMeta.CurrentTerm != meta2.CurrentTerm {
		t.Errorf("Expected term %d, got %d", meta2.CurrentTerm, loadedMeta.CurrentTerm)
	}

	if loadedMeta.VotedFor != meta2.VotedFor {
		t.Errorf("Expected votedFor %s, got %s", meta2.VotedFor, loadedMeta.VotedFor)
	}
}
