package raft

import (
	"fmt"
	"sync"

	raftpb "github.com/therealutkarshpriyadarshi/keyval/proto"
)

// Log provides thread-safe access to the Raft log
// This is a wrapper around the log slice in ServerState with additional helper methods
type Log struct {
	mu      sync.RWMutex
	entries []LogEntry
}

// NewLog creates a new empty log
func NewLog() *Log {
	return &Log{
		entries: make([]LogEntry, 0),
	}
}

// Append appends a new entry to the log
// Returns the index of the newly appended entry
func (l *Log) Append(entry LogEntry) uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, entry)
	return entry.Index
}

// AppendEntries appends multiple entries to the log
func (l *Log) AppendEntries(entries []LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = append(l.entries, entries...)
}

// Get retrieves a log entry by index
// Returns error if index is out of bounds
// Note: Log indices are 1-indexed as per Raft paper
func (l *Log) Get(index uint64) (LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index == 0 {
		return LogEntry{}, fmt.Errorf("invalid index: log indices start at 1")
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index) - 1

	if arrayIndex < 0 || arrayIndex >= len(l.entries) {
		return LogEntry{}, fmt.Errorf("index %d out of bounds (log size: %d)", index, len(l.entries))
	}

	return l.entries[arrayIndex], nil
}

// GetEntries returns a slice of log entries from startIndex to endIndex (inclusive)
// Returns error if indices are invalid
func (l *Log) GetEntries(startIndex, endIndex uint64) ([]LogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if startIndex == 0 {
		return nil, fmt.Errorf("invalid start index: log indices start at 1")
	}

	if startIndex > endIndex {
		return nil, fmt.Errorf("start index %d > end index %d", startIndex, endIndex)
	}

	// Convert 1-indexed to 0-indexed
	startArrayIndex := int(startIndex) - 1
	endArrayIndex := int(endIndex) - 1

	if startArrayIndex < 0 || endArrayIndex >= len(l.entries) {
		return nil, fmt.Errorf("indices out of bounds")
	}

	// Make a copy to avoid external modification
	result := make([]LogEntry, endArrayIndex-startArrayIndex+1)
	copy(result, l.entries[startArrayIndex:endArrayIndex+1])

	return result, nil
}

// GetFrom returns all log entries starting from index (inclusive)
// Returns empty slice if index is out of bounds
func (l *Log) GetFrom(index uint64) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index == 0 || index > uint64(len(l.entries)) {
		return []LogEntry{}
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index) - 1

	result := make([]LogEntry, len(l.entries)-arrayIndex)
	copy(result, l.entries[arrayIndex:])

	return result
}

// LastIndex returns the index of the last entry in the log
// Returns 0 if the log is empty
func (l *Log) LastIndex() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return 0
	}

	return l.entries[len(l.entries)-1].Index
}

// LastTerm returns the term of the last entry in the log
// Returns 0 if the log is empty
func (l *Log) LastTerm() uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return 0
	}

	return l.entries[len(l.entries)-1].Term
}

// LastIndexAndTerm returns both the index and term of the last entry
// Returns (0, 0) if the log is empty
func (l *Log) LastIndexAndTerm() (uint64, uint64) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return 0, 0
	}

	lastEntry := l.entries[len(l.entries)-1]
	return lastEntry.Index, lastEntry.Term
}

// Size returns the number of entries in the log
func (l *Log) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.entries)
}

// IsEmpty returns true if the log is empty
func (l *Log) IsEmpty() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return len(l.entries) == 0
}

// TruncateBefore removes all entries before (and including) the given index
// This is used for log compaction after creating a snapshot
func (l *Log) TruncateBefore(index uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if index == 0 {
		// Nothing to truncate
		return nil
	}

	// Find the position of the index
	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index)

	if arrayIndex >= len(l.entries) {
		// Truncating everything
		l.entries = make([]LogEntry, 0)
		return nil
	}

	// Keep entries after the index
	l.entries = l.entries[arrayIndex:]
	return nil
}

// TruncateAfter removes all entries after the given index (exclusive)
// This is used when resolving log conflicts
func (l *Log) TruncateAfter(index uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if index == 0 {
		// Truncate everything
		l.entries = make([]LogEntry, 0)
		return nil
	}

	if index > uint64(len(l.entries)) {
		return fmt.Errorf("truncate index %d out of bounds (log size: %d)", index, len(l.entries))
	}

	// Convert 1-indexed to 0-indexed and truncate
	arrayIndex := int(index)
	l.entries = l.entries[:arrayIndex]

	return nil
}

// GetTermAtIndex returns the term of the entry at the given index
// Returns 0 if index is out of bounds or 0
func (l *Log) GetTermAtIndex(index uint64) uint64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index == 0 || index > uint64(len(l.entries)) {
		return 0
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index) - 1
	return l.entries[arrayIndex].Term
}

// IsUpToDate checks if a candidate's log is at least as up-to-date as this log
// This implements the log comparison logic from the Raft paper (§5.4.1)
// A log is more up-to-date if:
// 1. The last entry has a higher term, OR
// 2. The last entries have the same term AND this log is at least as long
func (l *Log) IsUpToDate(candidateLastIndex, candidateLastTerm uint64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	lastIndex, lastTerm := l.lastIndexAndTermUnsafe()

	// If candidate's last log term is higher, their log is more up-to-date
	if candidateLastTerm > lastTerm {
		return true
	}

	// If terms are equal, the longer log is more up-to-date
	if candidateLastTerm == lastTerm {
		return candidateLastIndex >= lastIndex
	}

	// Candidate's last log term is lower, so their log is not up-to-date
	return false
}

// MatchesTerm checks if the entry at index has the given term
// Returns false if index is out of bounds
func (l *Log) MatchesTerm(index, term uint64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if index == 0 {
		// Index 0 is always "matched" with term 0 (represents empty log before first entry)
		return term == 0
	}

	if index > uint64(len(l.entries)) {
		return false
	}

	// Convert 1-indexed to 0-indexed
	arrayIndex := int(index) - 1
	return l.entries[arrayIndex].Term == term
}

// lastIndexAndTermUnsafe returns the last index and term without locking
// Must be called with lock held
func (l *Log) lastIndexAndTermUnsafe() (uint64, uint64) {
	if len(l.entries) == 0 {
		return 0, 0
	}

	lastEntry := l.entries[len(l.entries)-1]
	return lastEntry.Index, lastEntry.Term
}

// ConvertToProto converts internal log entries to protobuf format
func ConvertToProto(entries []LogEntry) []*raftpb.LogEntry {
	result := make([]*raftpb.LogEntry, len(entries))
	for i, entry := range entries {
		result[i] = &raftpb.LogEntry{
			Term:  entry.Term,
			Index: entry.Index,
			Type:  ConvertEntryTypeToProto(entry.Type),
			Data:  entry.Data,
		}
	}
	return result
}

// ConvertFromProto converts protobuf log entries to internal format
func ConvertFromProto(entries []*raftpb.LogEntry) []LogEntry {
	result := make([]LogEntry, len(entries))
	for i, entry := range entries {
		result[i] = LogEntry{
			Term:  entry.Term,
			Index: entry.Index,
			Type:  ConvertEntryTypeFromProto(entry.Type),
			Data:  entry.Data,
		}
	}
	return result
}

// ConvertEntryTypeToProto converts internal EntryType to protobuf
func ConvertEntryTypeToProto(t EntryType) raftpb.EntryType {
	switch t {
	case EntryNormal:
		return raftpb.EntryType_ENTRY_TYPE_NORMAL
	case EntryConfig:
		return raftpb.EntryType_ENTRY_TYPE_CONFIGURATION
	case EntryNoOp:
		return raftpb.EntryType_ENTRY_TYPE_NOOP
	default:
		return raftpb.EntryType_ENTRY_TYPE_NORMAL
	}
}

// ConvertEntryTypeFromProto converts protobuf EntryType to internal
func ConvertEntryTypeFromProto(t raftpb.EntryType) EntryType {
	switch t {
	case raftpb.EntryType_ENTRY_TYPE_NORMAL:
		return EntryNormal
	case raftpb.EntryType_ENTRY_TYPE_CONFIGURATION:
		return EntryConfig
	case raftpb.EntryType_ENTRY_TYPE_NOOP:
		return EntryNoOp
	default:
		return EntryNormal
	}
}

// GetAllEntries returns a copy of all entries in the log
func (l *Log) GetAllEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]LogEntry, len(l.entries))
	copy(result, l.entries)
	return result
}
