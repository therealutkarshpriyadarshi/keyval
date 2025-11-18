package rpc

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/therealutkarshpriyadarshi/keyval/proto"
	"google.golang.org/grpc"
)

// RaftHandler defines the interface for handling Raft RPCs
// The Raft node will implement this interface
type RaftHandler interface {
	HandleRequestVote(ctx context.Context, req *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error)
	HandleAppendEntries(ctx context.Context, req *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error)
	HandleInstallSnapshot(ctx context.Context, req *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error)
}

// Server represents a gRPC server for handling Raft RPCs
type Server struct {
	raftpb.UnimplementedRaftServiceServer
	mu       sync.RWMutex
	address  string
	listener net.Listener
	grpc     *grpc.Server
	handler  RaftHandler
	started  bool
}

// NewServer creates a new RPC server
func NewServer(address string, handler RaftHandler) (*Server, error) {
	if handler == nil {
		return nil, fmt.Errorf("handler cannot be nil")
	}

	s := &Server{
		address: address,
		handler: handler,
	}

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}

	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	s.listener = listener
	s.grpc = grpc.NewServer()

	// Register the Raft service
	raftpb.RegisterRaftServiceServer(s.grpc, s)

	s.started = true

	// Start serving in a goroutine
	go func() {
		if err := s.grpc.Serve(listener); err != nil {
			// Log error if needed
		}
	}()

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return nil
	}

	if s.grpc != nil {
		s.grpc.GracefulStop()
		s.grpc = nil
	}

	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}

	s.started = false
	return nil
}

// Address returns the server address
func (s *Server) Address() string {
	return s.address
}

// RequestVote implements the RequestVote RPC
func (s *Server) RequestVote(ctx context.Context, req *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error) {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	return handler.HandleRequestVote(ctx, req)
}

// AppendEntries implements the AppendEntries RPC
func (s *Server) AppendEntries(ctx context.Context, req *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error) {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	return handler.HandleAppendEntries(ctx, req)
}

// InstallSnapshot implements the InstallSnapshot RPC
func (s *Server) InstallSnapshot(ctx context.Context, req *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error) {
	s.mu.RLock()
	handler := s.handler
	s.mu.RUnlock()

	if handler == nil {
		return nil, fmt.Errorf("no handler configured")
	}

	return handler.HandleInstallSnapshot(ctx, req)
}
