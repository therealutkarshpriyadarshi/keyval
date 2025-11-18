package bench

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
)

func BenchmarkWAL_Append(b *testing.B) {
	dir := b.TempDir()
	wal, err := storage.NewWAL(dir)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	entry := &storage.WALEntry{
		Type:  storage.WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("benchmark data"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Index = uint64(i + 1)
		if err := wal.Append(entry); err != nil {
			b.Fatalf("Failed to append: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkWAL_AppendWithSync(b *testing.B) {
	dir := b.TempDir()
	wal, err := storage.NewWAL(dir)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	entry := &storage.WALEntry{
		Type:  storage.WALEntryLog,
		Index: 1,
		Term:  1,
		Data:  []byte("benchmark data"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Index = uint64(i + 1)
		if err := wal.Append(entry); err != nil {
			b.Fatalf("Failed to append: %v", err)
		}
		if err := wal.Sync(); err != nil {
			b.Fatalf("Failed to sync: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkWAL_AppendBatch(b *testing.B) {
	dir := b.TempDir()
	wal, err := storage.NewWAL(dir)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", size), func(b *testing.B) {
			entries := make([]*storage.WALEntry, size)
			for i := 0; i < size; i++ {
				entries[i] = &storage.WALEntry{
					Type:  storage.WALEntryLog,
					Index: uint64(i + 1),
					Term:  1,
					Data:  []byte("benchmark data"),
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := wal.AppendBatch(entries); err != nil {
					b.Fatalf("Failed to append batch: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkWAL_Read(b *testing.B) {
	dir := b.TempDir()
	wal, err := storage.NewWAL(dir)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	// Pre-populate WAL
	numEntries := 1000
	for i := 0; i < numEntries; i++ {
		entry := &storage.WALEntry{
			Type:  storage.WALEntryLog,
			Index: uint64(i + 1),
			Term:  1,
			Data:  []byte("benchmark data"),
		}
		if err := wal.Append(entry); err != nil {
			b.Fatalf("Failed to append: %v", err)
		}
	}
	if err := wal.Sync(); err != nil {
		b.Fatalf("Failed to sync: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wal.ReadAll(); err != nil {
			b.Fatalf("Failed to read: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkBoltStore_SaveMetadata(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	store, err := storage.NewBoltStore(path)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	meta := &storage.Metadata{
		CurrentTerm: 1,
		VotedFor:    "node-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		meta.CurrentTerm = uint64(i + 1)
		if err := store.SaveMetadata(meta); err != nil {
			b.Fatalf("Failed to save metadata: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkBoltStore_SaveLogEntry(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	store, err := storage.NewBoltStore(path)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	entry := &storage.LogEntry{
		Index: 1,
		Term:  1,
		Data:  []byte("benchmark log entry data"),
		Type:  0, // EntryNormal
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Index = uint64(i + 1)
		if err := store.SaveLogEntry(entry); err != nil {
			b.Fatalf("Failed to save log entry: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkBoltStore_SaveLogEntries(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	store, err := storage.NewBoltStore(path)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	batchSizes := []int{10, 100, 1000}

	for _, size := range batchSizes {
		b.Run(fmt.Sprintf("BatchSize%d", size), func(b *testing.B) {
			entries := make([]*storage.LogEntry, size)
			for i := 0; i < size; i++ {
				entries[i] = &storage.LogEntry{
					Index: uint64(i + 1),
					Term:  1,
					Data:  []byte("benchmark data"),
					Type:  0, // EntryNormal
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Update indices for each iteration
				for j := range entries {
					entries[j].Index = uint64(i*size + j + 1)
				}
				if err := store.SaveLogEntries(entries); err != nil {
					b.Fatalf("Failed to save log entries: %v", err)
				}
			}
			b.StopTimer()
		})
	}
}

func BenchmarkBoltStore_LoadLogEntry(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	store, err := storage.NewBoltStore(path)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Pre-populate store
	numEntries := 10000
	entries := make([]*storage.LogEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		entries[i] = &storage.LogEntry{
			Index: uint64(i + 1),
			Term:  1,
			Data:  []byte("benchmark data"),
			Type:  0, // EntryNormal
		}
	}
	if err := store.SaveLogEntries(entries); err != nil {
		b.Fatalf("Failed to save entries: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := uint64((i % numEntries) + 1)
		if _, err := store.LoadLogEntry(index); err != nil {
			b.Fatalf("Failed to load log entry: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkBoltStore_LoadAllLogEntries(b *testing.B) {
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.db")
	store, err := storage.NewBoltStore(path)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	entryCounts := []int{100, 1000, 10000}

	for _, count := range entryCounts {
		b.Run(fmt.Sprintf("Entries%d", count), func(b *testing.B) {
			// Pre-populate store
			entries := make([]*storage.LogEntry, count)
			for i := 0; i < count; i++ {
				entries[i] = &storage.LogEntry{
					Index: uint64(i + 1),
					Term:  1,
					Data:  []byte("benchmark data"),
					Type:  0, // EntryNormal
				}
			}
			if err := store.SaveLogEntries(entries); err != nil {
				b.Fatalf("Failed to save entries: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := store.LoadAllLogEntries(); err != nil {
					b.Fatalf("Failed to load all log entries: %v", err)
				}
			}
			b.StopTimer()

			// Clean up for next iteration
			if err := store.DeleteLogEntriesFrom(1); err != nil {
				b.Fatalf("Failed to delete entries: %v", err)
			}
		})
	}
}

func BenchmarkPersistence_PersistLogEntry(b *testing.B) {
	dir := b.TempDir()
	storePath := filepath.Join(dir, "bench.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := raft.NewPersistenceLayer(store, wal)

	entry := &storage.LogEntry{
		Index: 1,
		Term:  1,
		Data:  []byte("benchmark data"),
		Type:  0, // EntryNormal
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Index = uint64(i + 1)
		if err := persistence.PersistLogEntry(entry); err != nil {
			b.Fatalf("Failed to persist log entry: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkPersistence_LoadLogEntries(b *testing.B) {
	dir := b.TempDir()
	storePath := filepath.Join(dir, "bench.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := raft.NewPersistenceLayer(store, wal)

	// Pre-populate
	numEntries := 1000
	entries := make([]*storage.LogEntry, numEntries)
	for i := 0; i < numEntries; i++ {
		entries[i] = &storage.LogEntry{
			Index: uint64(i + 1),
			Term:  1,
			Data:  []byte("benchmark data"),
			Type:  0, // EntryNormal
		}
	}
	if err := persistence.PersistLogEntries(entries); err != nil {
		b.Fatalf("Failed to persist entries: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := persistence.LoadLogEntries(); err != nil {
			b.Fatalf("Failed to load log entries: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkRecovery_CreateCheckpoint(b *testing.B) {
	dir := b.TempDir()
	storePath := filepath.Join(dir, "bench.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := raft.NewPersistenceLayer(store, wal)
	recovery := raft.NewRecoveryManager(persistence)

	node, err := raft.NewNode("node-1", "localhost:5001", []raft.Peer{})
	if err != nil {
		b.Fatalf("Failed to create node: %v", err)
	}

	// Add some log entries
	for i := 1; i <= 100; i++ {
		node.AppendLog(&storage.LogEntry{
			Index: uint64(i),
			Term:  1,
			Data:  []byte("test data"),
			Type:  0, // EntryNormal
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := recovery.CreateCheckpoint(node); err != nil {
			b.Fatalf("Failed to create checkpoint: %v", err)
		}
	}
	b.StopTimer()
}

func BenchmarkRecovery_VerifyIntegrity(b *testing.B) {
	dir := b.TempDir()
	storePath := filepath.Join(dir, "bench.db")
	walPath := filepath.Join(dir, "wal")

	store, err := storage.NewBoltStore(storePath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	wal, err := storage.NewWAL(walPath)
	if err != nil {
		b.Fatalf("Failed to create WAL: %v", err)
	}
	defer wal.Close()

	persistence := raft.NewPersistenceLayer(store, wal)
	recovery := raft.NewRecoveryManager(persistence)

	// Populate with data
	entries := make([]*storage.LogEntry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = &storage.LogEntry{
			Index: uint64(i + 1),
			Term:  1,
			Data:  []byte("benchmark data"),
			Type:  0, // EntryNormal
		}
	}
	if err := persistence.PersistLogEntries(entries); err != nil {
		b.Fatalf("Failed to persist entries: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := recovery.VerifyIntegrity(); err != nil {
			b.Fatalf("Integrity verification failed: %v", err)
		}
	}
	b.StopTimer()
}
