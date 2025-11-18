package raft

import (
	"sync"
	"testing"
)

func TestNewLog(t *testing.T) {
	log := NewLog()
	if log == nil {
		t.Fatal("NewLog() returned nil")
	}

	if !log.IsEmpty() {
		t.Errorf("New log should be empty")
	}

	if log.Size() != 0 {
		t.Errorf("New log size should be 0, got %d", log.Size())
	}

	lastIndex, lastTerm := log.LastIndexAndTerm()
	if lastIndex != 0 || lastTerm != 0 {
		t.Errorf("Empty log should have lastIndex=0, lastTerm=0, got (%d, %d)", lastIndex, lastTerm)
	}
}

func TestLogAppend(t *testing.T) {
	log := NewLog()

	entry1 := LogEntry{Term: 1, Index: 1, Data: []byte("command1"), Type: EntryNormal}
	index := log.Append(entry1)

	if index != 1 {
		t.Errorf("Expected index 1, got %d", index)
	}

	if log.Size() != 1 {
		t.Errorf("Log size should be 1, got %d", log.Size())
	}

	entry2 := LogEntry{Term: 1, Index: 2, Data: []byte("command2"), Type: EntryNormal}
	index = log.Append(entry2)

	if index != 2 {
		t.Errorf("Expected index 2, got %d", index)
	}

	if log.Size() != 2 {
		t.Errorf("Log size should be 2, got %d", log.Size())
	}
}

func TestLogGet(t *testing.T) {
	log := NewLog()

	// Test getting from empty log
	_, err := log.Get(1)
	if err == nil {
		t.Error("Getting from empty log should return error")
	}

	// Test getting with index 0
	_, err = log.Get(0)
	if err == nil {
		t.Error("Getting with index 0 should return error")
	}

	// Add some entries
	entry1 := LogEntry{Term: 1, Index: 1, Data: []byte("command1"), Type: EntryNormal}
	entry2 := LogEntry{Term: 1, Index: 2, Data: []byte("command2"), Type: EntryNormal}
	entry3 := LogEntry{Term: 2, Index: 3, Data: []byte("command3"), Type: EntryNormal}

	log.Append(entry1)
	log.Append(entry2)
	log.Append(entry3)

	// Test getting valid entries
	got, err := log.Get(1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got.Index != 1 || got.Term != 1 {
		t.Errorf("Got wrong entry: %+v", got)
	}

	got, err = log.Get(2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got.Index != 2 {
		t.Errorf("Got wrong entry: %+v", got)
	}

	got, err = log.Get(3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if got.Index != 3 || got.Term != 2 {
		t.Errorf("Got wrong entry: %+v", got)
	}

	// Test getting out of bounds
	_, err = log.Get(4)
	if err == nil {
		t.Error("Getting out of bounds should return error")
	}
}

func TestLogLastIndexAndTerm(t *testing.T) {
	log := NewLog()

	// Empty log
	lastIndex, lastTerm := log.LastIndexAndTerm()
	if lastIndex != 0 || lastTerm != 0 {
		t.Errorf("Empty log: expected (0, 0), got (%d, %d)", lastIndex, lastTerm)
	}

	// Add entry
	entry1 := LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal}
	log.Append(entry1)

	lastIndex, lastTerm = log.LastIndexAndTerm()
	if lastIndex != 1 || lastTerm != 1 {
		t.Errorf("After first entry: expected (1, 1), got (%d, %d)", lastIndex, lastTerm)
	}

	// Add more entries
	entry2 := LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal}
	entry3 := LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal}
	log.Append(entry2)
	log.Append(entry3)

	lastIndex, lastTerm = log.LastIndexAndTerm()
	if lastIndex != 3 || lastTerm != 2 {
		t.Errorf("After three entries: expected (3, 2), got (%d, %d)", lastIndex, lastTerm)
	}

	// Test separate methods
	if log.LastIndex() != 3 {
		t.Errorf("LastIndex() expected 3, got %d", log.LastIndex())
	}
	if log.LastTerm() != 2 {
		t.Errorf("LastTerm() expected 2, got %d", log.LastTerm())
	}
}

func TestLogIsUpToDate(t *testing.T) {
	log := NewLog()

	// Empty log - any candidate is up-to-date
	if !log.IsUpToDate(0, 0) {
		t.Error("Empty log should consider (0, 0) up-to-date")
	}
	if !log.IsUpToDate(1, 1) {
		t.Error("Empty log should consider any non-empty log up-to-date")
	}

	// Add entries: [term=1, index=1], [term=1, index=2], [term=2, index=3]
	log.Append(LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal})
	log.Append(LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal})
	log.Append(LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal})

	tests := []struct {
		name              string
		candidateIndex    uint64
		candidateTerm     uint64
		expectedUpToDate  bool
		reason            string
	}{
		{"Same as current", 3, 2, true, "Same index and term should be up-to-date"},
		{"Higher term", 2, 3, true, "Higher term is always more up-to-date"},
		{"Lower term", 5, 1, false, "Lower term is never up-to-date"},
		{"Same term, longer log", 4, 2, true, "Same term but longer log is up-to-date"},
		{"Same term, shorter log", 2, 2, false, "Same term but shorter log is not up-to-date"},
		{"Empty candidate log", 0, 0, false, "Empty log is not up-to-date when current log has entries"},
		{"Higher term, empty index", 0, 3, true, "Higher term is up-to-date regardless of index"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := log.IsUpToDate(tt.candidateIndex, tt.candidateTerm)
			if result != tt.expectedUpToDate {
				t.Errorf("%s: expected %v, got %v. Reason: %s",
					tt.name, tt.expectedUpToDate, result, tt.reason)
			}
		})
	}
}

func TestLogMatchesTerm(t *testing.T) {
	log := NewLog()

	// Index 0 always matches term 0
	if !log.MatchesTerm(0, 0) {
		t.Error("Index 0 should match term 0")
	}

	// Add entries
	log.Append(LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal})
	log.Append(LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal})
	log.Append(LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal})

	tests := []struct {
		index    uint64
		term     uint64
		expected bool
	}{
		{0, 0, true},   // Index 0 matches term 0
		{1, 1, true},   // Correct match
		{2, 1, true},   // Correct match
		{3, 2, true},   // Correct match
		{1, 2, false},  // Wrong term
		{2, 2, false},  // Wrong term
		{3, 1, false},  // Wrong term
		{4, 1, false},  // Out of bounds
		{10, 5, false}, // Out of bounds
	}

	for _, tt := range tests {
		result := log.MatchesTerm(tt.index, tt.term)
		if result != tt.expected {
			t.Errorf("MatchesTerm(%d, %d) = %v, expected %v", tt.index, tt.term, result, tt.expected)
		}
	}
}

func TestLogTruncateAfter(t *testing.T) {
	log := NewLog()

	// Add entries
	for i := uint64(1); i <= 5; i++ {
		log.Append(LogEntry{Term: 1, Index: i, Data: []byte("cmd"), Type: EntryNormal})
	}

	if log.Size() != 5 {
		t.Fatalf("Expected size 5, got %d", log.Size())
	}

	// Truncate after index 3 (keep 1, 2, 3)
	err := log.TruncateAfter(3)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if log.Size() != 3 {
		t.Errorf("After truncate(3), expected size 3, got %d", log.Size())
	}

	lastIndex := log.LastIndex()
	if lastIndex != 3 {
		t.Errorf("After truncate(3), last index should be 3, got %d", lastIndex)
	}

	// Truncate everything
	err = log.TruncateAfter(0)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !log.IsEmpty() {
		t.Errorf("After truncate(0), log should be empty")
	}
}

func TestLogGetEntries(t *testing.T) {
	log := NewLog()

	// Add entries
	for i := uint64(1); i <= 5; i++ {
		log.Append(LogEntry{Term: 1, Index: i, Data: []byte("cmd"), Type: EntryNormal})
	}

	// Test valid range
	entries, err := log.GetEntries(2, 4)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}
	if entries[0].Index != 2 || entries[2].Index != 4 {
		t.Errorf("Wrong entries returned")
	}

	// Test invalid ranges
	_, err = log.GetEntries(0, 2)
	if err == nil {
		t.Error("GetEntries(0, 2) should return error")
	}

	_, err = log.GetEntries(4, 2)
	if err == nil {
		t.Error("GetEntries(4, 2) should return error (start > end)")
	}

	_, err = log.GetEntries(1, 10)
	if err == nil {
		t.Error("GetEntries with out of bounds end should return error")
	}
}

func TestLogGetFrom(t *testing.T) {
	log := NewLog()

	// Add entries
	for i := uint64(1); i <= 5; i++ {
		log.Append(LogEntry{Term: 1, Index: i, Data: []byte("cmd"), Type: EntryNormal})
	}

	// Test valid index
	entries := log.GetFrom(3)
	if len(entries) != 3 {
		t.Errorf("GetFrom(3) expected 3 entries, got %d", len(entries))
	}
	if entries[0].Index != 3 || entries[2].Index != 5 {
		t.Errorf("GetFrom(3) returned wrong entries")
	}

	// Test from beginning
	entries = log.GetFrom(1)
	if len(entries) != 5 {
		t.Errorf("GetFrom(1) expected 5 entries, got %d", len(entries))
	}

	// Test invalid indices
	entries = log.GetFrom(0)
	if len(entries) != 0 {
		t.Errorf("GetFrom(0) should return empty slice")
	}

	entries = log.GetFrom(10)
	if len(entries) != 0 {
		t.Errorf("GetFrom(10) should return empty slice")
	}
}

func TestLogGetTermAtIndex(t *testing.T) {
	log := NewLog()

	// Add entries with different terms
	log.Append(LogEntry{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal})
	log.Append(LogEntry{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal})
	log.Append(LogEntry{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal})
	log.Append(LogEntry{Term: 3, Index: 4, Data: []byte("cmd4"), Type: EntryNormal})

	tests := []struct {
		index        uint64
		expectedTerm uint64
	}{
		{0, 0},  // Out of bounds
		{1, 1},  // Valid
		{2, 1},  // Valid
		{3, 2},  // Valid
		{4, 3},  // Valid
		{5, 0},  // Out of bounds
		{10, 0}, // Out of bounds
	}

	for _, tt := range tests {
		term := log.GetTermAtIndex(tt.index)
		if term != tt.expectedTerm {
			t.Errorf("GetTermAtIndex(%d) = %d, expected %d", tt.index, term, tt.expectedTerm)
		}
	}
}

func TestLogAppendEntries(t *testing.T) {
	log := NewLog()

	entries := []LogEntry{
		{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryNormal},
		{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNormal},
	}

	log.AppendEntries(entries)

	if log.Size() != 3 {
		t.Errorf("Expected size 3, got %d", log.Size())
	}

	lastIndex, lastTerm := log.LastIndexAndTerm()
	if lastIndex != 3 || lastTerm != 2 {
		t.Errorf("Expected last (3, 2), got (%d, %d)", lastIndex, lastTerm)
	}
}

func TestLogConcurrentAccess(t *testing.T) {
	log := NewLog()

	// Pre-populate with some entries
	for i := uint64(1); i <= 10; i++ {
		log.Append(LogEntry{Term: 1, Index: i, Data: []byte("cmd"), Type: EntryNormal})
	}

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				log.LastIndexAndTerm()
				log.Size()
				log.IsEmpty()
				log.Get(5)
				log.MatchesTerm(5, 1)
			}
		}()
	}

	// Concurrent writes
	// Note: We use a counter starting from a known point to avoid index conflicts
	nextIndexCounter := uint64(11) // Start after the pre-populated 10 entries
	var indexMu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				indexMu.Lock()
				index := nextIndexCounter
				nextIndexCounter++
				indexMu.Unlock()

				log.Append(LogEntry{Term: 2, Index: index, Data: []byte("new"), Type: EntryNormal})
			}
		}()
	}

	wg.Wait()

	// Verify log is still consistent
	if log.IsEmpty() {
		t.Error("Log should not be empty after concurrent operations")
	}

	size := log.Size()
	lastIndex := log.LastIndex()

	// We should have at least the initial 10 entries
	if size < 10 {
		t.Errorf("Expected at least 10 entries, got %d", size)
	}

	// Last index should be at least the size (since we start with sequential indices)
	// With concurrent writes, lastIndex >= size (1-indexed vs 0-indexed array)
	if lastIndex < uint64(size) {
		t.Errorf("Last index (%d) should be at least size (%d)", lastIndex, size)
	}
}

func TestLogGetAllEntries(t *testing.T) {
	log := NewLog()

	// Empty log
	entries := log.GetAllEntries()
	if len(entries) != 0 {
		t.Errorf("GetAllEntries on empty log should return empty slice")
	}

	// Add some entries
	for i := uint64(1); i <= 5; i++ {
		log.Append(LogEntry{Term: 1, Index: i, Data: []byte("cmd"), Type: EntryNormal})
	}

	entries = log.GetAllEntries()
	if len(entries) != 5 {
		t.Errorf("Expected 5 entries, got %d", len(entries))
	}

	// Verify it's a copy (modifying returned slice shouldn't affect log)
	entries[0].Term = 999
	entry, _ := log.Get(1)
	if entry.Term == 999 {
		t.Error("GetAllEntries should return a copy, not a reference")
	}
}

func TestConvertToFromProto(t *testing.T) {
	entries := []LogEntry{
		{Term: 1, Index: 1, Data: []byte("cmd1"), Type: EntryNormal},
		{Term: 1, Index: 2, Data: []byte("cmd2"), Type: EntryConfig},
		{Term: 2, Index: 3, Data: []byte("cmd3"), Type: EntryNoOp},
	}

	// Convert to proto
	protoEntries := ConvertToProto(entries)
	if len(protoEntries) != 3 {
		t.Errorf("Expected 3 proto entries, got %d", len(protoEntries))
	}

	// Convert back
	converted := ConvertFromProto(protoEntries)
	if len(converted) != 3 {
		t.Errorf("Expected 3 converted entries, got %d", len(converted))
	}

	// Verify round-trip conversion
	for i := range entries {
		if entries[i].Term != converted[i].Term {
			t.Errorf("Entry %d: term mismatch", i)
		}
		if entries[i].Index != converted[i].Index {
			t.Errorf("Entry %d: index mismatch", i)
		}
		if entries[i].Type != converted[i].Type {
			t.Errorf("Entry %d: type mismatch", i)
		}
		if string(entries[i].Data) != string(converted[i].Data) {
			t.Errorf("Entry %d: data mismatch", i)
		}
	}
}
