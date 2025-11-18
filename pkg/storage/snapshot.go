package storage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Snapshot represents a point-in-time snapshot of the state machine
type Snapshot struct {
	Metadata SnapshotMetadata
	Data     []byte
}

// SnapshotManager manages snapshot storage and retrieval
type SnapshotManager struct {
	mu sync.RWMutex

	// Directory where snapshots are stored
	dir string

	// Current snapshot metadata
	currentSnapshot *SnapshotMetadata

	// Maximum number of snapshots to keep
	maxSnapshots int
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(dir string, maxSnapshots int) (*SnapshotManager, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	sm := &SnapshotManager{
		dir:          dir,
		maxSnapshots: maxSnapshots,
	}

	// Load the latest snapshot metadata
	if err := sm.loadLatestMetadata(); err != nil {
		return nil, fmt.Errorf("failed to load latest snapshot metadata: %w", err)
	}

	return sm, nil
}

// Create creates a new snapshot
// The data should be the serialized state machine state
func (sm *SnapshotManager) Create(index, term uint64, data []byte) (*SnapshotMetadata, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Calculate checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))

	// Create metadata
	metadata := &SnapshotMetadata{
		Index:     index,
		Term:      term,
		Size:      int64(len(data)),
		Checksum:  checksum,
		CreatedAt: time.Now(),
	}

	// Write snapshot atomically using temp file + rename
	tempPath := sm.snapshotPath(index, term) + ".tmp"
	finalPath := sm.snapshotPath(index, term)

	// Write data to temporary file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write snapshot data: %w", err)
	}

	// Write metadata to temporary file
	metadataPath := finalPath + ".meta"
	tempMetadataPath := metadataPath + ".tmp"
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(tempMetadataPath, metadataBytes, 0644); err != nil {
		os.Remove(tempPath)
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Atomically rename both files
	if err := os.Rename(tempPath, finalPath); err != nil {
		os.Remove(tempPath)
		os.Remove(tempMetadataPath)
		return nil, fmt.Errorf("failed to rename snapshot file: %w", err)
	}

	if err := os.Rename(tempMetadataPath, metadataPath); err != nil {
		os.Remove(tempMetadataPath)
		return nil, fmt.Errorf("failed to rename metadata file: %w", err)
	}

	// Update current snapshot
	sm.currentSnapshot = metadata

	// Clean up old snapshots
	if err := sm.cleanupOldSnapshots(); err != nil {
		// Log error but don't fail - cleanup is not critical
		fmt.Printf("Warning: failed to cleanup old snapshots: %v\n", err)
	}

	return metadata, nil
}

// Load loads the latest snapshot
func (sm *SnapshotManager) Load() (*Snapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentSnapshot == nil {
		return nil, fmt.Errorf("no snapshot available")
	}

	return sm.loadSnapshot(sm.currentSnapshot.Index, sm.currentSnapshot.Term)
}

// LoadAt loads a specific snapshot by index and term
func (sm *SnapshotManager) LoadAt(index, term uint64) (*Snapshot, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.loadSnapshot(index, term)
}

// GetLatestMetadata returns the metadata of the latest snapshot
func (sm *SnapshotManager) GetLatestMetadata() *SnapshotMetadata {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.currentSnapshot == nil {
		return nil
	}

	// Return a copy
	metadata := *sm.currentSnapshot
	return &metadata
}

// List returns metadata for all available snapshots
func (sm *SnapshotManager) List() ([]*SnapshotMetadata, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot directory: %w", err)
	}

	var snapshots []*SnapshotMetadata
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".meta" {
			continue
		}

		metadataPath := filepath.Join(sm.dir, entry.Name())
		metadata, err := sm.loadMetadataFromFile(metadataPath)
		if err != nil {
			continue // Skip corrupted metadata files
		}

		snapshots = append(snapshots, metadata)
	}

	return snapshots, nil
}

// OpenSnapshotWriter creates a writer for streaming snapshot data
func (sm *SnapshotManager) OpenSnapshotWriter(index, term uint64) (*SnapshotWriter, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	tempPath := sm.snapshotPath(index, term) + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot file: %w", err)
	}

	return &SnapshotWriter{
		file:      file,
		tempPath:  tempPath,
		finalPath: sm.snapshotPath(index, term),
		index:     index,
		term:      term,
		hash:      newSHA256HashWriter(),
		sm:        sm,
	}, nil
}

// snapshotPath returns the path to a snapshot file
func (sm *SnapshotManager) snapshotPath(index, term uint64) string {
	filename := fmt.Sprintf("snapshot-%d-%d.snap", index, term)
	return filepath.Join(sm.dir, filename)
}

// loadSnapshot loads a snapshot from disk
func (sm *SnapshotManager) loadSnapshot(index, term uint64) (*Snapshot, error) {
	path := sm.snapshotPath(index, term)
	metadataPath := path + ".meta"

	// Load metadata
	metadata, err := sm.loadMetadataFromFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	// Load data
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot file: %w", err)
	}

	// Verify checksum
	checksum := fmt.Sprintf("%x", sha256.Sum256(data))
	if checksum != metadata.Checksum {
		return nil, fmt.Errorf("snapshot checksum mismatch: expected %s, got %s",
			metadata.Checksum, checksum)
	}

	return &Snapshot{
		Metadata: *metadata,
		Data:     data,
	}, nil
}

// loadMetadataFromFile loads snapshot metadata from a file
func (sm *SnapshotManager) loadMetadataFromFile(path string) (*SnapshotMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var metadata SnapshotMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// loadLatestMetadata finds and loads the latest snapshot metadata
func (sm *SnapshotManager) loadLatestMetadata() error {
	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No snapshots yet
		}
		return err
	}

	var latest *SnapshotMetadata
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".meta" {
			continue
		}

		metadataPath := filepath.Join(sm.dir, entry.Name())
		metadata, err := sm.loadMetadataFromFile(metadataPath)
		if err != nil {
			continue // Skip corrupted metadata files
		}

		if latest == nil || metadata.Index > latest.Index {
			latest = metadata
		}
	}

	sm.currentSnapshot = latest
	return nil
}

// cleanupOldSnapshots removes old snapshots, keeping only the most recent ones
func (sm *SnapshotManager) cleanupOldSnapshots() error {
	snapshots, err := sm.List()
	if err != nil {
		return err
	}

	// Sort by index (descending)
	for i := 0; i < len(snapshots); i++ {
		for j := i + 1; j < len(snapshots); j++ {
			if snapshots[j].Index > snapshots[i].Index {
				snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
			}
		}
	}

	// Remove old snapshots
	for i := sm.maxSnapshots; i < len(snapshots); i++ {
		snapshotPath := sm.snapshotPath(snapshots[i].Index, snapshots[i].Term)
		metadataPath := snapshotPath + ".meta"

		os.Remove(snapshotPath)
		os.Remove(metadataPath)
	}

	return nil
}

// SnapshotWriter provides streaming write interface for large snapshots
type SnapshotWriter struct {
	file      *os.File
	tempPath  string
	finalPath string
	index     uint64
	term      uint64
	hash      *sha256HashWriter
	size      int64
	sm        *SnapshotManager
}

// Write writes data to the snapshot
func (sw *SnapshotWriter) Write(p []byte) (n int, err error) {
	// Write to file
	n, err = sw.file.Write(p)
	if err != nil {
		return n, err
	}

	// Update hash
	sw.hash.Write(p[:n])
	sw.size += int64(n)

	return n, nil
}

// Close finalizes the snapshot
func (sw *SnapshotWriter) Close() error {
	// Close file
	if err := sw.file.Close(); err != nil {
		os.Remove(sw.tempPath)
		return err
	}

	// Calculate checksum
	checksum := sw.hash.Sum()

	// Create metadata
	metadata := &SnapshotMetadata{
		Index:     sw.index,
		Term:      sw.term,
		Size:      sw.size,
		Checksum:  checksum,
		CreatedAt: time.Now(),
	}

	// Write metadata
	metadataPath := sw.finalPath + ".meta"
	tempMetadataPath := metadataPath + ".tmp"
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		os.Remove(sw.tempPath)
		return err
	}

	if err := os.WriteFile(tempMetadataPath, metadataBytes, 0644); err != nil {
		os.Remove(sw.tempPath)
		return err
	}

	// Atomically rename files
	if err := os.Rename(sw.tempPath, sw.finalPath); err != nil {
		os.Remove(sw.tempPath)
		os.Remove(tempMetadataPath)
		return err
	}

	if err := os.Rename(tempMetadataPath, metadataPath); err != nil {
		os.Remove(tempMetadataPath)
		return err
	}

	// Update current snapshot
	sw.sm.mu.Lock()
	sw.sm.currentSnapshot = metadata
	sw.sm.mu.Unlock()

	return nil
}

// Abort aborts the snapshot write
func (sw *SnapshotWriter) Abort() error {
	sw.file.Close()
	return os.Remove(sw.tempPath)
}

// sha256HashWriter wraps a hash for computing checksums
type sha256HashWriter struct {
	data []byte
}

func newSHA256HashWriter() *sha256HashWriter {
	return &sha256HashWriter{
		data: make([]byte, 0),
	}
}

func (h *sha256HashWriter) Write(p []byte) (n int, err error) {
	h.data = append(h.data, p...)
	return len(p), nil
}

func (h *sha256HashWriter) Sum() string {
	return fmt.Sprintf("%x", sha256.Sum256(h.data))
}
