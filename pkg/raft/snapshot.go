package raft

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
)

// Snapshotting manages snapshot creation and application
type Snapshotting struct {
	mu sync.Mutex

	node *Node

	// Snapshot manager for storage
	snapshotManager *storage.SnapshotManager

	// Threshold for triggering snapshot (log size in bytes)
	logSizeThreshold int64

	// Minimum log entries before snapshot
	minLogEntries uint64

	// Time-based snapshot interval
	snapshotInterval time.Duration

	// Last snapshot time
	lastSnapshotTime time.Time

	// Flag to prevent concurrent snapshot creation
	creating bool
}

// NewSnapshotting creates a new snapshotting instance
func NewSnapshotting(node *Node, snapshotDir string, config *SnapshotConfig) (*Snapshotting, error) {
	if config == nil {
		config = DefaultSnapshotConfig()
	}

	snapshotManager, err := storage.NewSnapshotManager(snapshotDir, config.MaxSnapshots)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot manager: %w", err)
	}

	return &Snapshotting{
		node:             node,
		snapshotManager:  snapshotManager,
		logSizeThreshold: config.LogSizeThreshold,
		minLogEntries:    config.MinLogEntries,
		snapshotInterval: config.Interval,
		lastSnapshotTime: time.Now(),
	}, nil
}

// ShouldSnapshot checks if a snapshot should be created
func (s *Snapshotting) ShouldSnapshot() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.creating {
		return false
	}

	// Check log size threshold
	s.node.mu.RLock()
	logSize := s.node.log.Size()
	lastApplied := s.node.lastApplied
	s.node.mu.RUnlock()

	// Get latest snapshot metadata
	latestSnapshot := s.snapshotManager.GetLatestMetadata()
	var lastIncludedIndex uint64
	if latestSnapshot != nil {
		lastIncludedIndex = latestSnapshot.Index
	}

	// Check if we have enough new entries
	newEntries := lastApplied - lastIncludedIndex
	if newEntries < s.minLogEntries {
		return false
	}

	// Check log size threshold
	if s.logSizeThreshold > 0 && logSize >= s.logSizeThreshold {
		return true
	}

	// Check time-based interval
	if s.snapshotInterval > 0 && time.Since(s.lastSnapshotTime) >= s.snapshotInterval {
		return true
	}

	return false
}

// CreateSnapshot creates a snapshot of the current state
func (s *Snapshotting) CreateSnapshot() error {
	s.mu.Lock()
	if s.creating {
		s.mu.Unlock()
		return fmt.Errorf("snapshot creation already in progress")
	}
	s.creating = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.creating = false
		s.lastSnapshotTime = time.Now()
		s.mu.Unlock()
	}()

	// Get current state
	s.node.mu.RLock()
	lastApplied := s.node.lastApplied
	if lastApplied == 0 {
		s.node.mu.RUnlock()
		return fmt.Errorf("no entries have been applied yet")
	}

	// Get the term of the last applied entry
	lastEntry, err := s.node.log.Get(lastApplied)
	if err != nil {
		s.node.mu.RUnlock()
		return fmt.Errorf("failed to get last applied entry: %w", err)
	}
	lastTerm := lastEntry.Term

	// Create snapshot from state machine
	stateMachine := s.node.stateMachine
	s.node.mu.RUnlock()

	if stateMachine == nil {
		return fmt.Errorf("state machine not initialized")
	}

	// Create snapshot data
	snapshotData, err := stateMachine.Snapshot()
	if err != nil {
		return fmt.Errorf("failed to create snapshot from state machine: %w", err)
	}

	// Save snapshot to disk
	metadata, err := s.snapshotManager.Create(lastApplied, lastTerm, snapshotData)
	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	// Log snapshot creation
	s.node.logger.Printf("Created snapshot: index=%d, term=%d, size=%d bytes",
		metadata.Index, metadata.Term, metadata.Size)

	// Compact the log (remove entries up to lastApplied)
	if err := s.CompactLog(lastApplied); err != nil {
		s.node.logger.Printf("Warning: failed to compact log after snapshot: %v", err)
		// Don't fail - snapshot is still valid
	}

	return nil
}

// CompactLog truncates the log up to the given index
func (s *Snapshotting) CompactLog(index uint64) error {
	s.node.mu.Lock()
	defer s.node.mu.Unlock()

	// Truncate in-memory log
	if err := s.node.log.Truncate(index); err != nil {
		return fmt.Errorf("failed to truncate in-memory log: %w", err)
	}

	// Truncate persistent log (WAL)
	if s.node.store != nil {
		if err := s.node.store.DeleteLogEntriesFrom(0); err != nil {
			// Try to delete entries before the snapshot index
			// Note: This is storage-dependent
			s.node.logger.Printf("Warning: failed to delete log entries from storage: %v", err)
		}
	}

	return nil
}

// RestoreSnapshot restores the state machine from the latest snapshot
func (s *Snapshotting) RestoreSnapshot() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load latest snapshot
	snapshot, err := s.snapshotManager.Load()
	if err != nil {
		return fmt.Errorf("failed to load snapshot: %w", err)
	}

	// Restore state machine
	s.node.mu.Lock()
	defer s.node.mu.Unlock()

	if s.node.stateMachine == nil {
		return fmt.Errorf("state machine not initialized")
	}

	if err := s.node.stateMachine.Restore(snapshot.Data); err != nil {
		return fmt.Errorf("failed to restore state machine: %w", err)
	}

	// Update Raft state
	s.node.lastApplied = snapshot.Metadata.Index
	s.node.commitIndex = snapshot.Metadata.Index

	s.node.logger.Printf("Restored snapshot: index=%d, term=%d",
		snapshot.Metadata.Index, snapshot.Metadata.Term)

	return nil
}

// GetLatestSnapshotMetadata returns the metadata of the latest snapshot
func (s *Snapshotting) GetLatestSnapshotMetadata() *storage.SnapshotMetadata {
	return s.snapshotManager.GetLatestMetadata()
}

// LoadSnapshot loads a specific snapshot
func (s *Snapshotting) LoadSnapshot(index, term uint64) (*storage.Snapshot, error) {
	return s.snapshotManager.LoadAt(index, term)
}

// SnapshotConfig contains configuration for snapshotting
type SnapshotConfig struct {
	// Log size threshold in bytes
	LogSizeThreshold int64

	// Minimum log entries before snapshot
	MinLogEntries uint64

	// Time interval for periodic snapshots
	Interval time.Duration

	// Maximum number of snapshots to keep
	MaxSnapshots int
}

// DefaultSnapshotConfig returns default snapshot configuration
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		LogSizeThreshold: 10 * 1024 * 1024, // 10 MB
		MinLogEntries:    1000,              // At least 1000 entries
		Interval:         1 * time.Hour,     // Every hour
		MaxSnapshots:     3,                 // Keep 3 snapshots
	}
}

// BackgroundSnapshotChecker periodically checks if snapshot is needed
func (s *Snapshotting) BackgroundSnapshotChecker(stopCh <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.ShouldSnapshot() {
				if err := s.CreateSnapshot(); err != nil {
					s.node.logger.Printf("Background snapshot creation failed: %v", err)
				}
			}
		case <-stopCh:
			return
		}
	}
}
