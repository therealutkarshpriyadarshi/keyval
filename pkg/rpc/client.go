package rpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	raftpb "github.com/therealutkarshpriyadarshi/keyval/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client represents a gRPC client for communicating with Raft peers
type Client struct {
	mu      sync.RWMutex
	address string
	conn    *grpc.ClientConn
	client  raftpb.RaftServiceClient
	timeout time.Duration
}

// NewClient creates a new RPC client to a peer
func NewClient(address string, timeout time.Duration) (*Client, error) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	c := &Client{
		address: address,
		timeout: timeout,
	}

	return c, nil
}

// Connect establishes a connection to the peer
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil // Already connected
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.address, err)
	}

	c.conn = conn
	c.client = raftpb.NewRaftServiceClient(conn)
	return nil
}

// Close closes the connection to the peer
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.client = nil
		return err
	}
	return nil
}

// RequestVote sends a RequestVote RPC to the peer
func (c *Client) RequestVote(ctx context.Context, req *raftpb.RequestVoteRequest) (*raftpb.RequestVoteResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.RequestVote(ctx, req)
}

// AppendEntries sends an AppendEntries RPC to the peer
func (c *Client) AppendEntries(ctx context.Context, req *raftpb.AppendEntriesRequest) (*raftpb.AppendEntriesResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.AppendEntries(ctx, req)
}

// InstallSnapshot sends an InstallSnapshot RPC to the peer
func (c *Client) InstallSnapshot(ctx context.Context, req *raftpb.InstallSnapshotRequest) (*raftpb.InstallSnapshotResponse, error) {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return client.InstallSnapshot(ctx, req)
}

// Address returns the address of the peer
func (c *Client) Address() string {
	return c.address
}

// ClientPool manages connections to multiple peers
type ClientPool struct {
	mu      sync.RWMutex
	clients map[string]*Client
	timeout time.Duration
}

// NewClientPool creates a new client pool
func NewClientPool(timeout time.Duration) *ClientPool {
	return &ClientPool{
		clients: make(map[string]*Client),
		timeout: timeout,
	}
}

// GetClient returns a client for the given peer address, creating one if necessary
func (p *ClientPool) GetClient(address string) (*Client, error) {
	p.mu.RLock()
	client, exists := p.clients[address]
	p.mu.RUnlock()

	if exists {
		return client, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	client, exists = p.clients[address]
	if exists {
		return client, nil
	}

	// Create new client
	client, err := NewClient(address, p.timeout)
	if err != nil {
		return nil, err
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	p.clients[address] = client
	return client, nil
}

// RemoveClient removes a client from the pool
func (p *ClientPool) RemoveClient(address string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, exists := p.clients[address]
	if !exists {
		return nil
	}

	delete(p.clients, address)
	return client.Close()
}

// CloseAll closes all clients in the pool
func (p *ClientPool) CloseAll() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}

	p.clients = make(map[string]*Client)
	return lastErr
}
