package storage

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	// SegmentSizeBytes is the maximum size of a WAL segment (64MB)
	SegmentSizeBytes = 64 * 1024 * 1024

	// WALFilePattern is the naming pattern for WAL files
	WALFilePattern = "wal-%016d.log"
)

// FileWAL is a file-based implementation of the WAL interface
type FileWAL struct {
	dir          string
	mu           sync.RWMutex
	currentFile  *os.File
	currentIndex uint64
	writer       *bufio.Writer
	segmentSize  int64
}

// NewWAL creates a new write-ahead log
func NewWAL(dir string) (*FileWAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	wal := &FileWAL{
		dir:         dir,
		segmentSize: SegmentSizeBytes,
	}

	// Find the latest segment or create a new one
	segments, err := wal.listSegments()
	if err != nil {
		return nil, err
	}

	if len(segments) > 0 {
		// Open the last segment
		lastSegment := segments[len(segments)-1]
		wal.currentIndex = lastSegment
		if err := wal.openSegment(lastSegment); err != nil {
			return nil, err
		}
	} else {
		// Create the first segment
		if err := wal.createSegment(0); err != nil {
			return nil, err
		}
	}

	return wal, nil
}

// Append appends an entry to the WAL
func (w *FileWAL) Append(entry *WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.appendEntry(entry)
}

// AppendBatch appends multiple entries atomically
func (w *FileWAL) AppendBatch(entries []*WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, entry := range entries {
		if err := w.appendEntry(entry); err != nil {
			return err
		}
	}

	return w.sync()
}

// appendEntry is the internal append method (must be called with lock held)
func (w *FileWAL) appendEntry(entry *WALEntry) error {
	// Calculate checksum
	entry.Checksum = entry.CalculateChecksum()
	entry.Timestamp = time.Now()

	// Serialize the entry
	data, err := w.serializeEntry(entry)
	if err != nil {
		return fmt.Errorf("failed to serialize entry: %w", err)
	}

	// Check if we need to rotate to a new segment
	if w.shouldRotate(int64(len(data))) {
		if err := w.rotateSegment(); err != nil {
			return fmt.Errorf("failed to rotate segment: %w", err)
		}
	}

	// Write the entry
	if _, err := w.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// Read reads entries starting from the given index
func (w *FileWAL) Read(startIndex uint64) ([]*WALEntry, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	segments, err := w.listSegments()
	if err != nil {
		return nil, err
	}

	var entries []*WALEntry
	for _, segIdx := range segments {
		segEntries, err := w.readSegment(segIdx)
		if err != nil {
			return nil, err
		}

		for _, entry := range segEntries {
			if entry.Index >= startIndex {
				entries = append(entries, entry)
			}
		}
	}

	return entries, nil
}

// ReadAll reads all entries from the WAL
func (w *FileWAL) ReadAll() ([]*WALEntry, error) {
	return w.Read(0)
}

// Sync forces a sync to disk
func (w *FileWAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.sync()
}

// sync is the internal sync method (must be called with lock held)
func (w *FileWAL) sync() error {
	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return fmt.Errorf("failed to flush writer: %w", err)
		}
	}

	if w.currentFile != nil {
		if err := w.currentFile.Sync(); err != nil {
			return fmt.Errorf("failed to sync file: %w", err)
		}
	}

	return nil
}

// Truncate removes entries after the given index
func (w *FileWAL) Truncate(index uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Read all entries (without lock - we already have it)
	segments, err := w.listSegments()
	if err != nil {
		return err
	}

	var allEntries []*WALEntry
	for _, segIdx := range segments {
		segEntries, err := w.readSegment(segIdx)
		if err != nil {
			return err
		}
		allEntries = append(allEntries, segEntries...)
	}

	// Filter entries
	var keepEntries []*WALEntry
	for _, entry := range allEntries {
		if entry.Index <= index {
			keepEntries = append(keepEntries, entry)
		}
	}

	// Close current file
	if err := w.closeCurrentFile(); err != nil {
		return err
	}

	// Remove all segments
	segments, listErr := w.listSegments()
	if listErr != nil {
		return listErr
	}

	for _, segIdx := range segments {
		path := w.segmentPath(segIdx)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove segment: %w", err)
		}
	}

	// Create new segment and write kept entries
	if err := w.createSegment(0); err != nil {
		return err
	}

	for _, entry := range keepEntries {
		if err := w.appendEntry(entry); err != nil {
			return err
		}
	}

	return w.sync()
}

// Close closes the WAL
func (w *FileWAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.closeCurrentFile()
}

// Helper methods

func (w *FileWAL) shouldRotate(nextEntrySize int64) bool {
	if w.currentFile == nil {
		return false
	}

	info, err := w.currentFile.Stat()
	if err != nil {
		return false
	}

	return info.Size()+nextEntrySize > w.segmentSize
}

func (w *FileWAL) rotateSegment() error {
	if err := w.closeCurrentFile(); err != nil {
		return err
	}

	w.currentIndex++
	return w.createSegment(w.currentIndex)
}

func (w *FileWAL) createSegment(index uint64) error {
	path := w.segmentPath(index)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create segment: %w", err)
	}

	w.currentFile = file
	w.currentIndex = index
	w.writer = bufio.NewWriter(file)

	return nil
}

func (w *FileWAL) openSegment(index uint64) error {
	path := w.segmentPath(index)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open segment: %w", err)
	}

	w.currentFile = file
	w.currentIndex = index
	w.writer = bufio.NewWriter(file)

	return nil
}

func (w *FileWAL) closeCurrentFile() error {
	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return err
		}
		w.writer = nil
	}

	if w.currentFile != nil {
		if err := w.currentFile.Close(); err != nil {
			return err
		}
		w.currentFile = nil
	}

	return nil
}

func (w *FileWAL) segmentPath(index uint64) string {
	filename := fmt.Sprintf(WALFilePattern, index)
	return filepath.Join(w.dir, filename)
}

func (w *FileWAL) listSegments() ([]uint64, error) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL directory: %w", err)
	}

	var segments []uint64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		var index uint64
		if _, err := fmt.Sscanf(entry.Name(), WALFilePattern, &index); err == nil {
			segments = append(segments, index)
		}
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i] < segments[j]
	})

	return segments, nil
}

func (w *FileWAL) readSegment(index uint64) ([]*WALEntry, error) {
	path := w.segmentPath(index)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open segment: %w", err)
	}
	defer file.Close()

	var entries []*WALEntry
	reader := bufio.NewReader(file)

	for {
		entry, err := w.deserializeEntry(reader)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize entry: %w", err)
		}

		// Validate checksum
		if !entry.ValidateChecksum() {
			return nil, fmt.Errorf("checksum validation failed for entry at index %d", entry.Index)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (w *FileWAL) serializeEntry(entry *WALEntry) ([]byte, error) {
	// Format: [size:4][type:1][index:8][term:8][timestamp:8][dataLen:4][data][checksum:4]
	dataLen := len(entry.Data)
	totalSize := 1 + 8 + 8 + 8 + 4 + dataLen + 4

	buf := make([]byte, 4+totalSize)
	binary.BigEndian.PutUint32(buf[0:4], uint32(totalSize))

	offset := 4
	buf[offset] = byte(entry.Type)
	offset++

	binary.BigEndian.PutUint64(buf[offset:offset+8], entry.Index)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:offset+8], entry.Term)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(entry.Timestamp.Unix()))
	offset += 8

	binary.BigEndian.PutUint32(buf[offset:offset+4], uint32(dataLen))
	offset += 4

	copy(buf[offset:offset+dataLen], entry.Data)
	offset += dataLen

	binary.BigEndian.PutUint32(buf[offset:offset+4], entry.Checksum)

	return buf, nil
}

func (w *FileWAL) deserializeEntry(reader *bufio.Reader) (*WALEntry, error) {
	// Read size
	sizeBuf := make([]byte, 4)
	if _, err := io.ReadFull(reader, sizeBuf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(sizeBuf)

	// Read entry data
	data := make([]byte, size)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, err
	}

	offset := 0
	entry := &WALEntry{}

	entry.Type = WALEntryType(data[offset])
	offset++

	entry.Index = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	entry.Term = binary.BigEndian.Uint64(data[offset : offset+8])
	offset += 8

	timestamp := binary.BigEndian.Uint64(data[offset : offset+8])
	entry.Timestamp = time.Unix(int64(timestamp), 0)
	offset += 8

	dataLen := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	entry.Data = make([]byte, dataLen)
	copy(entry.Data, data[offset:offset+int(dataLen)])
	offset += int(dataLen)

	entry.Checksum = binary.BigEndian.Uint32(data[offset : offset+4])

	return entry, nil
}
