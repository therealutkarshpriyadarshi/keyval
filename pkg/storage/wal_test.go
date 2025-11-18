package storage

import (
	"testing"
)

func TestWAL_NewWAL(t *testing.T) {
	dir := t.TempDir()

	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	if wal.dir != dir {
		t.Errorf("Expected dir %s, got %s", dir, wal.dir)
	}
}

func TestWAL_Append(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	entry := &WALEntry{
		Type:  WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("test data"),
	}

	if err := wal.Append(entry); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}
}

func TestWAL_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// Append multiple entries
	entries := []*WALEntry{
		{Type: WALEntryLog, Index: 1, Term: 1, Data: []byte("entry 1")},
		{Type: WALEntryLog, Index: 2, Term: 1, Data: []byte("entry 2")},
		{Type: WALEntryLog, Index: 3, Term: 2, Data: []byte("entry 3")},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Close and reopen
	wal.Close()

	wal, err = NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	// Read all entries
	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(readEntries))
	}

	for i, entry := range readEntries {
		if entry.Index != entries[i].Index {
			t.Errorf("Entry %d: expected index %d, got %d", i, entries[i].Index, entry.Index)
		}
		if entry.Term != entries[i].Term {
			t.Errorf("Entry %d: expected term %d, got %d", i, entries[i].Term, entry.Term)
		}
		if string(entry.Data) != string(entries[i].Data) {
			t.Errorf("Entry %d: expected data %s, got %s", i, entries[i].Data, entry.Data)
		}
	}
}

func TestWAL_AppendBatch(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	entries := []*WALEntry{
		{Type: WALEntryLog, Index: 1, Term: 1, Data: []byte("batch 1")},
		{Type: WALEntryLog, Index: 2, Term: 1, Data: []byte("batch 2")},
		{Type: WALEntryLog, Index: 3, Term: 1, Data: []byte("batch 3")},
	}

	if err := wal.AppendBatch(entries); err != nil {
		t.Fatalf("Failed to append batch: %v", err)
	}

	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(readEntries))
	}
}

func TestWAL_Truncate(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append entries
	entries := []*WALEntry{
		{Type: WALEntryLog, Index: 1, Term: 1, Data: []byte("entry 1")},
		{Type: WALEntryLog, Index: 2, Term: 1, Data: []byte("entry 2")},
		{Type: WALEntryLog, Index: 3, Term: 2, Data: []byte("entry 3")},
		{Type: WALEntryLog, Index: 4, Term: 2, Data: []byte("entry 4")},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Truncate after index 2
	if err := wal.Truncate(2); err != nil {
		t.Fatalf("Failed to truncate: %v", err)
	}

	// Read all entries
	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != 2 {
		t.Fatalf("Expected 2 entries after truncate, got %d", len(readEntries))
	}

	for i, entry := range readEntries {
		if entry.Index != entries[i].Index {
			t.Errorf("Entry %d: expected index %d, got %d", i, entries[i].Index, entry.Index)
		}
	}
}

func TestWAL_Read(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Append entries
	entries := []*WALEntry{
		{Type: WALEntryLog, Index: 1, Term: 1, Data: []byte("entry 1")},
		{Type: WALEntryLog, Index: 2, Term: 1, Data: []byte("entry 2")},
		{Type: WALEntryLog, Index: 3, Term: 2, Data: []byte("entry 3")},
		{Type: WALEntryLog, Index: 4, Term: 2, Data: []byte("entry 4")},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Read from index 2
	readEntries, err := wal.Read(2)
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != 3 {
		t.Fatalf("Expected 3 entries, got %d", len(readEntries))
	}

	if readEntries[0].Index != 2 {
		t.Errorf("Expected first entry index 2, got %d", readEntries[0].Index)
	}
}

func TestWAL_ChecksumValidation(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	entry := &WALEntry{
		Type:  WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("test data"),
	}

	if err := wal.Append(entry); err != nil {
		t.Fatalf("Failed to append entry: %v", err)
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	wal.Close()

	// Reopen and read
	wal, err = NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(readEntries))
	}

	if !readEntries[0].ValidateChecksum() {
		t.Error("Checksum validation failed")
	}
}

func TestWAL_SegmentRotation(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Set small segment size for testing
	wal.segmentSize = 512 // 512 bytes

	// Append entries that exceed segment size
	largeData := make([]byte, 400) // 400 bytes per entry
	for i := 1; i <= 10; i++ {
		entry := &WALEntry{
			Type:  WALEntryLog,
			Index: uint64(i),
			Term:  1,
			Data:  largeData,
		}

		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append entry %d: %v", i, err)
		}
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Check that multiple segments were created
	segments, err := wal.listSegments()
	if err != nil {
		t.Fatalf("Failed to list segments: %v", err)
	}

	if len(segments) < 2 {
		t.Errorf("Expected at least 2 segments, got %d", len(segments))
	}
}

func TestWAL_ConcurrentAppend(t *testing.T) {
	dir := t.TempDir()
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Concurrent appends
	done := make(chan bool)
	for i := 1; i <= 10; i++ {
		go func(index int) {
			entry := &WALEntry{
				Type:  WALEntryLog,
				Index: uint64(index),
				Term:  1,
				Data:  []byte("concurrent entry"),
			}

			if err := wal.Append(entry); err != nil {
				t.Errorf("Failed to append entry %d: %v", index, err)
			}
			done <- true
		}(i)
	}

	// Wait for all appends
	for i := 0; i < 10; i++ {
		<-done
	}

	if err := wal.Sync(); err != nil {
		t.Fatalf("Failed to sync: %v", err)
	}

	// Read all entries
	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != 10 {
		t.Errorf("Expected 10 entries, got %d", len(readEntries))
	}
}

func TestWALEntry_CalculateChecksum(t *testing.T) {
	entry := &WALEntry{
		Type:  WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("test"),
	}

	checksum1 := entry.CalculateChecksum()
	checksum2 := entry.CalculateChecksum()

	if checksum1 != checksum2 {
		t.Error("Checksum should be deterministic")
	}

	// Modify data
	entry.Data = []byte("modified")
	checksum3 := entry.CalculateChecksum()

	if checksum1 == checksum3 {
		t.Error("Checksum should change when data changes")
	}
}

func TestWALEntry_ValidateChecksum(t *testing.T) {
	entry := &WALEntry{
		Type:  WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("test"),
	}

	entry.Checksum = entry.CalculateChecksum()

	if !entry.ValidateChecksum() {
		t.Error("Valid checksum should pass validation")
	}

	// Corrupt checksum
	entry.Checksum = 12345

	if entry.ValidateChecksum() {
		t.Error("Invalid checksum should fail validation")
	}
}

func TestWAL_Persistence(t *testing.T) {
	dir := t.TempDir()

	// Create and write entries
	wal, err := NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	entries := []*WALEntry{
		{Type: WALEntryLog, Index: 1, Term: 1, Data: []byte("entry 1")},
		{Type: WALEntryMetadata, Index: 0, Term: 2, Data: []byte("metadata")},
	}

	for _, entry := range entries {
		if err := wal.Append(entry); err != nil {
			t.Fatalf("Failed to append entry: %v", err)
		}
	}

	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Reopen and verify
	wal, err = NewWAL(dir)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}
	defer wal.Close()

	readEntries, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("Expected %d entries, got %d", len(entries), len(readEntries))
	}
}
