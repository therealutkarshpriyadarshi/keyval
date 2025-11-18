package raft

import (
	"encoding/json"
	"fmt"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
)

// PersistenceLayer handles persistence of Raft state
type PersistenceLayer struct {
	store storage.Store
	wal   storage.WAL
}

// NewPersistenceLayer creates a new persistence layer
func NewPersistenceLayer(store storage.Store, wal storage.WAL) *PersistenceLayer {
	return &PersistenceLayer{
		store: store,
		wal:   wal,
	}
}

// PersistMetadata persists current term and votedFor to stable storage
// This must be called before responding to RequestVote RPCs
func (p *PersistenceLayer) PersistMetadata(term uint64, votedFor string) error {
	meta := &storage.Metadata{
		CurrentTerm: term,
		VotedFor:    votedFor,
	}

	if err := p.store.SaveMetadata(meta); err != nil {
		return fmt.Errorf("failed to persist metadata: %w", err)
	}

	// Also write to WAL for additional durability
	walEntry := &storage.WALEntry{
		Type:  storage.WALEntryMetadata,
		Index: 0,
		Term:  term,
		Data:  []byte(votedFor),
	}

	if err := p.wal.Append(walEntry); err != nil {
		return fmt.Errorf("failed to append metadata to WAL: %w", err)
	}

	return p.wal.Sync()
}

// LoadMetadata loads persisted metadata from stable storage
func (p *PersistenceLayer) LoadMetadata() (uint64, string, error) {
	meta, err := p.store.LoadMetadata()
	if err != nil {
		return 0, "", fmt.Errorf("failed to load metadata: %w", err)
	}

	return meta.CurrentTerm, meta.VotedFor, nil
}

// PersistLogEntry persists a log entry to stable storage
// This must be called before appending entries to the log
func (p *PersistenceLayer) PersistLogEntry(entry *LogEntry) error {
	return p.PersistLogEntries([]*LogEntry{entry})
}

// PersistLogEntries persists multiple log entries to stable storage
func (p *PersistenceLayer) PersistLogEntries(entries []*LogEntry) error {
	// Convert to storage.LogEntry
	storageEntries := make([]*storage.LogEntry, len(entries))
	for i, entry := range entries {
		storageEntries[i] = &storage.LogEntry{
			Index: entry.Index,
			Term:  entry.Term,
			Data:  entry.Data,
			Type:  int(entry.Type),
		}
	}

	// Save to BoltDB
	if err := p.store.SaveLogEntries(storageEntries); err != nil {
		return fmt.Errorf("failed to save log entries: %w", err)
	}

	// Write to WAL
	walEntries := make([]*storage.WALEntry, len(entries))
	for i, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal log entry: %w", err)
		}

		walEntries[i] = &storage.WALEntry{
			Type:  storage.WALEntryLog,
			Index: entry.Index,
			Term:  entry.Term,
			Data:  data,
		}
	}

	if err := p.wal.AppendBatch(walEntries); err != nil {
		return fmt.Errorf("failed to append entries to WAL: %w", err)
	}

	return p.wal.Sync()
}

// LoadLogEntries loads all log entries from stable storage
func (p *PersistenceLayer) LoadLogEntries() ([]*LogEntry, error) {
	storageEntries, err := p.store.LoadAllLogEntries()
	if err != nil {
		return nil, fmt.Errorf("failed to load log entries: %w", err)
	}

	// Convert from storage.LogEntry to raft.LogEntry
	entries := make([]*LogEntry, len(storageEntries))
	for i, entry := range storageEntries {
		entries[i] = &LogEntry{
			Index: entry.Index,
			Term:  entry.Term,
			Data:  entry.Data,
			Type:  EntryType(entry.Type),
		}
	}

	return entries, nil
}

// LoadLogEntriesRange loads log entries in a range [start, end]
func (p *PersistenceLayer) LoadLogEntriesRange(start, end uint64) ([]*LogEntry, error) {
	storageEntries, err := p.store.LoadLogEntries(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to load log entries range: %w", err)
	}

	// Convert from storage.LogEntry to raft.LogEntry
	entries := make([]*LogEntry, len(storageEntries))
	for i, entry := range storageEntries {
		entries[i] = &LogEntry{
			Index: entry.Index,
			Term:  entry.Term,
			Data:  entry.Data,
			Type:  EntryType(entry.Type),
		}
	}

	return entries, nil
}

// DeleteLogEntriesFrom deletes log entries from index onwards
func (p *PersistenceLayer) DeleteLogEntriesFrom(index uint64) error {
	if err := p.store.DeleteLogEntriesFrom(index); err != nil {
		return fmt.Errorf("failed to delete log entries: %w", err)
	}

	// Truncate WAL as well
	if err := p.wal.Truncate(index - 1); err != nil {
		return fmt.Errorf("failed to truncate WAL: %w", err)
	}

	return p.wal.Sync()
}

// GetFirstIndex returns the first log index
func (p *PersistenceLayer) GetFirstIndex() (uint64, error) {
	return p.store.FirstIndex()
}

// GetLastIndex returns the last log index
func (p *PersistenceLayer) GetLastIndex() (uint64, error) {
	return p.store.LastIndex()
}

// PersistSnapshot persists snapshot metadata
func (p *PersistenceLayer) PersistSnapshot(meta *storage.SnapshotMetadata) error {
	if err := p.store.SaveSnapshot(meta); err != nil {
		return fmt.Errorf("failed to save snapshot metadata: %w", err)
	}

	// Write snapshot metadata to WAL
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot metadata: %w", err)
	}

	walEntry := &storage.WALEntry{
		Type:  storage.WALEntrySnapshot,
		Index: meta.Index,
		Term:  meta.Term,
		Data:  data,
	}

	if err := p.wal.Append(walEntry); err != nil {
		return fmt.Errorf("failed to append snapshot metadata to WAL: %w", err)
	}

	return p.wal.Sync()
}

// LoadSnapshot loads snapshot metadata
func (p *PersistenceLayer) LoadSnapshot() (*storage.SnapshotMetadata, error) {
	meta, err := p.store.LoadSnapshot()
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot metadata: %w", err)
	}

	return meta, nil
}

// Close closes the persistence layer
func (p *PersistenceLayer) Close() error {
	var firstErr error

	if err := p.wal.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	if err := p.store.Close(); err != nil && firstErr == nil {
		firstErr = err
	}

	return firstErr
}

// Sync forces a sync to disk
func (p *PersistenceLayer) Sync() error {
	return p.wal.Sync()
}
