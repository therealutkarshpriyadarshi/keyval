package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotManager_Create(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	data := []byte("test snapshot data")
	metadata, err := sm.Create(10, 5, data)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	if metadata.Index != 10 {
		t.Errorf("Expected index 10, got %d", metadata.Index)
	}

	if metadata.Term != 5 {
		t.Errorf("Expected term 5, got %d", metadata.Term)
	}

	if metadata.Size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), metadata.Size)
	}

	if metadata.Checksum == "" {
		t.Error("Expected non-empty checksum")
	}
}

func TestSnapshotManager_LoadAndRestore(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Create snapshot
	originalData := []byte("test snapshot data for restoration")
	_, err = sm.Create(20, 10, originalData)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Load snapshot
	snapshot, err := sm.Load()
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	if !bytes.Equal(snapshot.Data, originalData) {
		t.Errorf("Snapshot data mismatch: expected %s, got %s", originalData, snapshot.Data)
	}

	if snapshot.Metadata.Index != 20 {
		t.Errorf("Expected index 20, got %d", snapshot.Metadata.Index)
	}

	if snapshot.Metadata.Term != 10 {
		t.Errorf("Expected term 10, got %d", snapshot.Metadata.Term)
	}
}

func TestSnapshotManager_LoadAt(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 5)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Create multiple snapshots
	data1 := []byte("snapshot 1")
	_, err = sm.Create(10, 1, data1)
	if err != nil {
		t.Fatalf("Failed to create snapshot 1: %v", err)
	}

	data2 := []byte("snapshot 2")
	_, err = sm.Create(20, 2, data2)
	if err != nil {
		t.Fatalf("Failed to create snapshot 2: %v", err)
	}

	// Load specific snapshot
	snapshot, err := sm.LoadAt(10, 1)
	if err != nil {
		t.Fatalf("Failed to load snapshot at (10,1): %v", err)
	}

	if !bytes.Equal(snapshot.Data, data1) {
		t.Errorf("Snapshot data mismatch for (10,1)")
	}

	snapshot, err = sm.LoadAt(20, 2)
	if err != nil {
		t.Fatalf("Failed to load snapshot at (20,2): %v", err)
	}

	if !bytes.Equal(snapshot.Data, data2) {
		t.Errorf("Snapshot data mismatch for (20,2)")
	}
}

func TestSnapshotManager_GetLatestMetadata(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// No snapshot yet
	if metadata := sm.GetLatestMetadata(); metadata != nil {
		t.Error("Expected nil metadata when no snapshots exist")
	}

	// Create first snapshot
	sm.Create(10, 1, []byte("data1"))

	metadata := sm.GetLatestMetadata()
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if metadata.Index != 10 {
		t.Errorf("Expected index 10, got %d", metadata.Index)
	}

	// Create newer snapshot
	sm.Create(20, 2, []byte("data2"))

	metadata = sm.GetLatestMetadata()
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}
	if metadata.Index != 20 {
		t.Errorf("Expected index 20 (latest), got %d", metadata.Index)
	}
}

func TestSnapshotManager_List(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 5)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Create multiple snapshots
	for i := 1; i <= 3; i++ {
		_, err = sm.Create(uint64(i*10), uint64(i), []byte("data"))
		if err != nil {
			t.Fatalf("Failed to create snapshot %d: %v", i, err)
		}
	}

	snapshots, err := sm.List()
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) != 3 {
		t.Errorf("Expected 3 snapshots, got %d", len(snapshots))
	}
}

func TestSnapshotManager_CleanupOldSnapshots(t *testing.T) {
	dir := t.TempDir()
	maxSnapshots := 3
	sm, err := NewSnapshotManager(dir, maxSnapshots)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Create more snapshots than the limit
	for i := 1; i <= 5; i++ {
		_, err = sm.Create(uint64(i*10), uint64(i), []byte("data"))
		if err != nil {
			t.Fatalf("Failed to create snapshot %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different creation times
	}

	// Check that old snapshots were cleaned up
	snapshots, err := sm.List()
	if err != nil {
		t.Fatalf("Failed to list snapshots: %v", err)
	}

	if len(snapshots) > maxSnapshots {
		t.Errorf("Expected at most %d snapshots after cleanup, got %d", maxSnapshots, len(snapshots))
	}

	// Verify the latest snapshot is still there
	latest := sm.GetLatestMetadata()
	if latest == nil || latest.Index != 50 {
		t.Error("Latest snapshot should be preserved")
	}
}

func TestSnapshotManager_ChecksumVerification(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Create snapshot
	data := []byte("test data for checksum")
	_, err = sm.Create(10, 1, data)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Corrupt the snapshot data
	snapshotPath := sm.snapshotPath(10, 1)
	corruptedData := []byte("corrupted data")
	if err := os.WriteFile(snapshotPath, corruptedData, 0644); err != nil {
		t.Fatalf("Failed to corrupt snapshot: %v", err)
	}

	// Try to load corrupted snapshot
	_, err = sm.LoadAt(10, 1)
	if err == nil {
		t.Error("Expected error when loading corrupted snapshot")
	}
}

func TestSnapshotWriter_StreamingWrite(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Open snapshot writer
	writer, err := sm.OpenSnapshotWriter(30, 5)
	if err != nil {
		t.Fatalf("Failed to open snapshot writer: %v", err)
	}

	// Write data in chunks
	chunks := [][]byte{
		[]byte("chunk 1 "),
		[]byte("chunk 2 "),
		[]byte("chunk 3"),
	}

	for _, chunk := range chunks {
		n, err := writer.Write(chunk)
		if err != nil {
			t.Fatalf("Failed to write chunk: %v", err)
		}
		if n != len(chunk) {
			t.Errorf("Expected to write %d bytes, wrote %d", len(chunk), n)
		}
	}

	// Close writer
	if err := writer.Close(); err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// Load and verify
	snapshot, err := sm.Load()
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	expectedData := []byte("chunk 1 chunk 2 chunk 3")
	if !bytes.Equal(snapshot.Data, expectedData) {
		t.Errorf("Expected data %s, got %s", expectedData, snapshot.Data)
	}
}

func TestSnapshotWriter_Abort(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Open snapshot writer
	writer, err := sm.OpenSnapshotWriter(40, 6)
	if err != nil {
		t.Fatalf("Failed to open snapshot writer: %v", err)
	}

	// Write some data
	writer.Write([]byte("test data"))

	// Abort the write
	if err := writer.Abort(); err != nil {
		t.Fatalf("Failed to abort writer: %v", err)
	}

	// Verify temp file was removed
	tempPath := writer.tempPath
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Error("Temp file should be removed after abort")
	}

	// Verify snapshot was not created
	if metadata := sm.GetLatestMetadata(); metadata != nil {
		t.Error("No snapshot should exist after abort")
	}
}

func TestSnapshotManager_RecoveryAfterRestart(t *testing.T) {
	dir := t.TempDir()

	// Create first manager and snapshot
	sm1, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create first snapshot manager: %v", err)
	}

	data := []byte("persistent data")
	_, err = sm1.Create(50, 10, data)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Create new manager (simulating restart)
	sm2, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create second snapshot manager: %v", err)
	}

	// Verify snapshot is available
	metadata := sm2.GetLatestMetadata()
	if metadata == nil {
		t.Fatal("Expected snapshot to be loaded after restart")
	}

	if metadata.Index != 50 || metadata.Term != 10 {
		t.Errorf("Expected snapshot (50,10), got (%d,%d)", metadata.Index, metadata.Term)
	}

	// Load and verify data
	snapshot, err := sm2.Load()
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	if !bytes.Equal(snapshot.Data, data) {
		t.Error("Snapshot data mismatch after restart")
	}
}

func TestSnapshotManager_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSnapshotManager(dir, 3)
	if err != nil {
		t.Fatalf("Failed to create snapshot manager: %v", err)
	}

	// Verify no .tmp files remain after successful creation
	data := []byte("test data")
	_, err = sm.Create(60, 12, data)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v", err)
	}

	// Check for temp files
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("Found temp file after successful snapshot: %s", entry.Name())
		}
	}
}
