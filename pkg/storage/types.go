package storage

import (
	"hash/crc32"
	"time"
)

// LogEntry represents a Raft log entry (defined here to avoid import cycles)
type LogEntry struct {
	Term  uint64
	Index uint64
	Data  []byte
	Type  int
}

// WALEntryType represents the type of WAL entry
type WALEntryType uint8

const (
	// WALEntryLog represents a log entry
	WALEntryLog WALEntryType = iota + 1
	// WALEntryMetadata represents metadata (term, votedFor)
	WALEntryMetadata
	// WALEntrySnapshot represents snapshot metadata
	WALEntrySnapshot
)

// WALEntry represents a single entry in the write-ahead log
type WALEntry struct {
	Type      WALEntryType
	Index     uint64
	Term      uint64
	Timestamp time.Time
	Data      []byte
	Checksum  uint32
}

// CalculateChecksum calculates CRC32 checksum for the entry
func (e *WALEntry) CalculateChecksum() uint32 {
	crc := crc32.NewIEEE()
	crc.Write([]byte{byte(e.Type)})
	crc.Write(uint64ToBytes(e.Index))
	crc.Write(uint64ToBytes(e.Term))
	crc.Write(e.Data)
	return crc.Sum32()
}

// ValidateChecksum validates the entry's checksum
func (e *WALEntry) ValidateChecksum() bool {
	return e.Checksum == e.CalculateChecksum()
}

// Metadata represents persistent Raft state
type Metadata struct {
	CurrentTerm uint64
	VotedFor    string
}

// SnapshotMetadata represents snapshot information
type SnapshotMetadata struct {
	// Index is the last included index in the snapshot (replaces LastIncludedIndex)
	Index uint64 `json:"index"`

	// Term is the term of the last included index (replaces LastIncludedTerm)
	Term uint64 `json:"term"`

	// Configuration is the cluster configuration at this point
	Configuration []byte `json:"configuration,omitempty"`

	// Size is the size of the snapshot data in bytes
	Size int64 `json:"size"`

	// Checksum is the SHA-256 checksum of the snapshot data (stored as string for JSON)
	Checksum string `json:"checksum"`

	// ChecksumCRC32 is the CRC32 checksum (for backwards compatibility)
	ChecksumCRC32 uint32 `json:"checksum_crc32,omitempty"`

	// CreatedAt is when the snapshot was created (replaces Timestamp)
	CreatedAt time.Time `json:"created_at"`
}

// WAL interface defines write-ahead log operations
type WAL interface {
	// Append appends an entry to the WAL
	Append(entry *WALEntry) error

	// AppendBatch appends multiple entries atomically
	AppendBatch(entries []*WALEntry) error

	// Read reads entries starting from the given index
	Read(startIndex uint64) ([]*WALEntry, error)

	// ReadAll reads all entries from the WAL
	ReadAll() ([]*WALEntry, error)

	// Sync forces a sync to disk
	Sync() error

	// Truncate removes entries after the given index
	Truncate(index uint64) error

	// Close closes the WAL
	Close() error
}

// Store interface defines persistent storage operations
type Store interface {
	// SaveMetadata saves Raft metadata (term, votedFor)
	SaveMetadata(meta *Metadata) error

	// LoadMetadata loads Raft metadata
	LoadMetadata() (*Metadata, error)

	// SaveLogEntry saves a log entry
	SaveLogEntry(entry *LogEntry) error

	// SaveLogEntries saves multiple log entries
	SaveLogEntries(entries []*LogEntry) error

	// LoadLogEntry loads a log entry by index
	LoadLogEntry(index uint64) (*LogEntry, error)

	// LoadLogEntries loads log entries in a range [start, end]
	LoadLogEntries(start, end uint64) ([]*LogEntry, error)

	// LoadAllLogEntries loads all log entries
	LoadAllLogEntries() ([]*LogEntry, error)

	// DeleteLogEntriesFrom deletes log entries from index onwards
	DeleteLogEntriesFrom(index uint64) error

	// FirstIndex returns the first log index
	FirstIndex() (uint64, error)

	// LastIndex returns the last log index
	LastIndex() (uint64, error)

	// SaveSnapshot saves snapshot metadata
	SaveSnapshot(meta *SnapshotMetadata) error

	// LoadSnapshot loads snapshot metadata
	LoadSnapshot() (*SnapshotMetadata, error)

	// Close closes the store
	Close() error
}

// Helper functions

func uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	b[0] = byte(v >> 56)
	b[1] = byte(v >> 48)
	b[2] = byte(v >> 40)
	b[3] = byte(v >> 32)
	b[4] = byte(v >> 24)
	b[5] = byte(v >> 16)
	b[6] = byte(v >> 8)
	b[7] = byte(v)
	return b
}

func bytesToUint64(b []byte) uint64 {
	return uint64(b[0])<<56 |
		uint64(b[1])<<48 |
		uint64(b[2])<<40 |
		uint64(b[3])<<32 |
		uint64(b[4])<<24 |
		uint64(b[5])<<16 |
		uint64(b[6])<<8 |
		uint64(b[7])
}
