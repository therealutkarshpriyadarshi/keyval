package storage

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"

	bolt "go.etcd.io/bbolt"
)

var (
	// Bucket names
	metadataBucket = []byte("metadata")
	logBucket      = []byte("log")
	snapshotBucket = []byte("snapshot")

	// Metadata keys
	currentTermKey = []byte("currentTerm")
	votedForKey    = []byte("votedFor")
)

// BoltStore is a BoltDB-based implementation of the Store interface
type BoltStore struct {
	db *bolt.DB
	mu sync.RWMutex
}

// NewBoltStore creates a new BoltDB store
func NewBoltStore(path string) (*BoltStore, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open BoltDB: %w", err)
	}

	// Create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(metadataBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(logBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(snapshotBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// SaveMetadata saves Raft metadata (term, votedFor)
func (s *BoltStore) SaveMetadata(meta *Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(metadataBucket)

		// Save currentTerm
		termBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(termBytes, meta.CurrentTerm)
		if err := b.Put(currentTermKey, termBytes); err != nil {
			return err
		}

		// Save votedFor
		if err := b.Put(votedForKey, []byte(meta.VotedFor)); err != nil {
			return err
		}

		return nil
	})
}

// LoadMetadata loads Raft metadata
func (s *BoltStore) LoadMetadata() (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta := &Metadata{}

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(metadataBucket)

		// Load currentTerm
		termBytes := b.Get(currentTermKey)
		if termBytes != nil {
			meta.CurrentTerm = binary.BigEndian.Uint64(termBytes)
		}

		// Load votedFor
		votedForBytes := b.Get(votedForKey)
		if votedForBytes != nil {
			meta.VotedFor = string(votedForBytes)
		}

		return nil
	})

	return meta, err
}

// SaveLogEntry saves a log entry
func (s *BoltStore) SaveLogEntry(entry *LogEntry) error {
	return s.SaveLogEntries([]*LogEntry{entry})
}

// SaveLogEntries saves multiple log entries
func (s *BoltStore) SaveLogEntries(entries []*LogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)

		for _, entry := range entries {
			data, err := json.Marshal(entry)
			if err != nil {
				return fmt.Errorf("failed to marshal log entry: %w", err)
			}

			key := uint64ToBytes(entry.Index)
			if err := b.Put(key, data); err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadLogEntry loads a log entry by index
func (s *BoltStore) LoadLogEntry(index uint64) (*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entry *LogEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		key := uint64ToBytes(index)
		data := b.Get(key)

		if data == nil {
			return fmt.Errorf("log entry not found at index %d", index)
		}

		entry = &LogEntry{}
		if err := json.Unmarshal(data, entry); err != nil {
			return fmt.Errorf("failed to unmarshal log entry: %w", err)
		}

		return nil
	})

	return entry, err
}

// LoadLogEntries loads log entries in a range [start, end]
func (s *BoltStore) LoadLogEntries(start, end uint64) ([]*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []*LogEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		c := b.Cursor()

		startKey := uint64ToBytes(start)
		endKey := uint64ToBytes(end)

		for k, v := c.Seek(startKey); k != nil && binary.BigEndian.Uint64(k) <= end; k, v = c.Next() {
			if binary.BigEndian.Uint64(k) > binary.BigEndian.Uint64(endKey) {
				break
			}

			entry := &LogEntry{}
			if err := json.Unmarshal(v, entry); err != nil {
				return fmt.Errorf("failed to unmarshal log entry: %w", err)
			}

			entries = append(entries, entry)
		}

		return nil
	})

	return entries, err
}

// LoadAllLogEntries loads all log entries
func (s *BoltStore) LoadAllLogEntries() ([]*LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var entries []*LogEntry

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			entry := &LogEntry{}
			if err := json.Unmarshal(v, entry); err != nil {
				return fmt.Errorf("failed to unmarshal log entry: %w", err)
			}

			entries = append(entries, entry)
		}

		return nil
	})

	return entries, err
}

// DeleteLogEntriesFrom deletes log entries from index onwards
func (s *BoltStore) DeleteLogEntriesFrom(index uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		c := b.Cursor()

		startKey := uint64ToBytes(index)

		// Collect keys to delete
		var keysToDelete [][]byte
		for k, _ := c.Seek(startKey); k != nil; k, _ = c.Next() {
			keyCopy := make([]byte, len(k))
			copy(keyCopy, k)
			keysToDelete = append(keysToDelete, keyCopy)
		}

		// Delete collected keys
		for _, key := range keysToDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

// FirstIndex returns the first log index
func (s *BoltStore) FirstIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var firstIndex uint64

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		c := b.Cursor()

		k, _ := c.First()
		if k != nil {
			firstIndex = binary.BigEndian.Uint64(k)
		}

		return nil
	})

	return firstIndex, err
}

// LastIndex returns the last log index
func (s *BoltStore) LastIndex() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var lastIndex uint64

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(logBucket)
		c := b.Cursor()

		k, _ := c.Last()
		if k != nil {
			lastIndex = binary.BigEndian.Uint64(k)
		}

		return nil
	})

	return lastIndex, err
}

// SaveSnapshot saves snapshot metadata
func (s *BoltStore) SaveSnapshot(meta *SnapshotMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(snapshotBucket)

		data, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("failed to marshal snapshot metadata: %w", err)
		}

		return b.Put([]byte("latest"), data)
	})
}

// LoadSnapshot loads snapshot metadata
func (s *BoltStore) LoadSnapshot() (*SnapshotMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var meta *SnapshotMetadata

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(snapshotBucket)
		data := b.Get([]byte("latest"))

		if data == nil {
			return nil // No snapshot
		}

		meta = &SnapshotMetadata{}
		if err := json.Unmarshal(data, meta); err != nil {
			return fmt.Errorf("failed to unmarshal snapshot metadata: %w", err)
		}

		return nil
	})

	return meta, err
}

// Close closes the store
func (s *BoltStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Close()
}
