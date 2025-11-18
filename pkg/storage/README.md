# Storage Package

Persistent storage layer with write-ahead logging for the KeyVal distributed database.

## Overview

The storage package provides durable persistence for Raft state with write-ahead logging (WAL),
snapshot management, and BoltDB-based key-value storage. It implements crash recovery, data
integrity verification with checksums, and efficient log compaction.

## Components

### Write-Ahead Log (WAL)
- **wal.go**: File-based WAL implementation with segment rotation
  - CRC32 checksums for data integrity
  - Automatic segment rotation (64MB default)
  - Fsync for durability guarantees
  - Concurrent read/write support
  - Efficient batch operations

### Persistent Store
- **store.go**: BoltDB interface for persistent state
  - Metadata storage (currentTerm, votedFor)
  - Log entry storage with efficient indexing
  - Snapshot metadata management
  - Batch write operations
  - Thread-safe concurrent access

### Snapshot Management
- **snapshot.go**: Snapshot creation, storage, and restoration
  - Atomic snapshot creation (temp + rename)
  - SHA-256 checksum verification
  - Automatic cleanup of old snapshots
  - Streaming snapshot writes for large datasets
  - Metadata tracking (index, term, size, checksum)

### Type Definitions
- **types.go**: Core types and interfaces
  - WAL interface and entry types
  - Store interface
  - Metadata and snapshot structures
  - Helper functions

## Features

✅ **Durability**: Write-ahead logging with fsync ensures no data loss
✅ **Integrity**: CRC32/SHA-256 checksums detect corruption
✅ **Performance**: Batched writes and efficient indexing
✅ **Crash Recovery**: Automatic state recovery on restart
✅ **Compaction**: Log size management via snapshots
✅ **Concurrency**: Thread-safe operations with RWMutex

## Usage Example

```go
// Create persistent storage
store, err := storage.NewBoltStore("/data/raft.db")
wal, err := storage.NewWAL("/data/wal")

// Create persistence layer
persistence := raft.NewPersistenceLayer(store, wal)

// Persist metadata
err = persistence.PersistMetadata(term, votedFor)

// Persist log entries
err = persistence.PersistLogEntries(entries)

// Force sync to disk
err = persistence.Sync()

// Crash recovery
term, votedFor, err := persistence.LoadMetadata()
entries, err := persistence.LoadLogEntries()
```

## Performance

Benchmarks on standard hardware:
- WAL append: ~0.05ms/op (buffered)
- WAL append with sync: ~0.5-1ms/op
- BoltDB metadata save: ~0.5-1ms/op
- BoltDB batch writes: ~2-5ms for 100 entries

## Testing

Comprehensive test coverage includes:
- Unit tests for all components
- Crash recovery simulation
- Concurrent access stress tests
- Checksum validation
- Segment rotation
- Snapshot integrity

Run tests:
```bash
go test ./pkg/storage/... -v
```

Run benchmarks:
```bash
go test ./pkg/storage/... -bench=. -benchtime=1s
```

## Status

✅ **Phase 4: COMPLETED** (Week 5)

All Phase 4 deliverables completed:
- ✅ WAL implementation with fsync and checksums
- ✅ BoltDB integration for persistent state
- ✅ Crash recovery mechanisms
- ✅ Data integrity verification
- ✅ Comprehensive test coverage
- ✅ Performance benchmarks
- ✅ Snapshot support (from Week 7, integrated early)
