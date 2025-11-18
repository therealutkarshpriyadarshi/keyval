package raft

import (
	"context"
	"fmt"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/storage"
	raftpb "github.com/therealutkarshpriyadarshi/keyval/proto"
)

// InstallSnapshot handles the InstallSnapshot RPC
// This is used when a follower is so far behind that the leader has discarded
// the log entries it needs. The leader sends its snapshot instead.
func (n *Node) InstallSnapshot(req *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	currentTerm := n.serverState.GetCurrentTerm()

	// Reply immediately if term < currentTerm
	if req.Term < currentTerm {
		return &raftpb.InstallSnapshotResponse{
			Term: currentTerm,
		}, nil
	}

	// Update term if we see a higher term
	if req.Term > currentTerm {
		n.serverState.SetCurrentTerm(req.Term)
		n.serverState.SetVotedFor("")
		n.becomeFollower(req.Term)
	}

	// Reset election timer since we heard from the leader
	n.electionTimer.Reset()

	// If this is the first chunk, create a new snapshot writer
	if req.Offset == 0 {
		// Initialize snapshot reception
		if n.snapshotWriter != nil {
			n.snapshotWriter.Abort()
		}

		writer, err := n.createSnapshotWriter(req.LastIncludedIndex, req.LastIncludedTerm)
		if err != nil {
			n.logger.Printf("Failed to create snapshot writer: %v", err)
			return &raftpb.InstallSnapshotResponse{
				Term: currentTerm,
			}, err
		}
		n.snapshotWriter = writer
	}

	// Write the chunk
	if n.snapshotWriter != nil {
		if _, err := n.snapshotWriter.Write(req.Data); err != nil {
			n.logger.Printf("Failed to write snapshot chunk: %v", err)
			n.snapshotWriter.Abort()
			n.snapshotWriter = nil
			return &raftpb.InstallSnapshotResponse{
				Term: currentTerm,
			}, err
		}
	}

	// If this is the last chunk, finalize the snapshot
	if req.Done {
		if n.snapshotWriter != nil {
			if err := n.snapshotWriter.Close(); err != nil {
				n.logger.Printf("Failed to close snapshot writer: %v", err)
				n.snapshotWriter.Abort()
				n.snapshotWriter = nil
				return &raftpb.InstallSnapshotResponse{
					Term: currentTerm,
				}, err
			}
			n.snapshotWriter = nil

			// Apply the snapshot to the state machine
			if err := n.applySnapshot(req.LastIncludedIndex, req.LastIncludedTerm); err != nil {
				n.logger.Printf("Failed to apply snapshot: %v", err)
				return &raftpb.InstallSnapshotResponse{
					Term: currentTerm,
				}, err
			}

			n.logger.Printf("Successfully installed snapshot: index=%d, term=%d",
				req.LastIncludedIndex, req.LastIncludedTerm)
		}
	}

	return &raftpb.InstallSnapshotResponse{
		Term: currentTerm,
	}, nil
}

// SendSnapshotToFollower sends a snapshot to a follower
func (n *Node) SendSnapshotToFollower(peerID string, snapshot *storage.Snapshot) error {
	peer, err := n.cluster.GetPeer(peerID)
	if err != nil {
		return fmt.Errorf("peer not found: %w", err)
	}

	// Get RPC client for the peer
	client, err := n.clientPool.GetClient(peer.Address)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	// Send snapshot in chunks
	chunkSize := 64 * 1024 // 64 KB chunks
	data := snapshot.Data
	offset := uint64(0)

	for offset < uint64(len(data)) {
		end := offset + uint64(chunkSize)
		if end > uint64(len(data)) {
			end = uint64(len(data))
		}

		chunk := data[offset:end]
		done := end >= uint64(len(data))

		req := &raftpb.InstallSnapshotRequest{
			Term:              n.serverState.GetCurrentTerm(),
			LeaderId:          n.id,
			LastIncludedIndex: snapshot.Metadata.Index,
			LastIncludedTerm:  snapshot.Metadata.Term,
			Offset:            offset,
			Data:              chunk,
			Done:              done,
		}

		resp, err := client.InstallSnapshot(context.Background(), req)
		if err != nil {
			return fmt.Errorf("failed to send snapshot chunk: %w", err)
		}

		// Check if follower has a higher term
		if resp.Term > req.Term {
			n.mu.Lock()
			n.serverState.SetCurrentTerm(resp.Term)
			n.becomeFollower(resp.Term)
			n.mu.Unlock()
			return fmt.Errorf("follower has higher term")
		}

		offset = end
	}

	n.logger.Printf("Successfully sent snapshot to %s: index=%d, term=%d",
		peerID, snapshot.Metadata.Index, snapshot.Metadata.Term)

	return nil
}

// createSnapshotWriter creates a writer for receiving snapshots
func (n *Node) createSnapshotWriter(index, term uint64) (*storage.SnapshotWriter, error) {
	// This would need to be implemented with access to the snapshot manager
	// For now, return an error indicating this needs to be wired up
	return nil, fmt.Errorf("snapshot writer creation not yet implemented")
}

// applySnapshot applies a received snapshot to the state machine
func (n *Node) applySnapshot(index, term uint64) error {
	// Load the snapshot that was just written
	// This would need access to the snapshot manager
	// For now, return an error indicating this needs to be wired up
	return fmt.Errorf("snapshot application not yet implemented")
}
