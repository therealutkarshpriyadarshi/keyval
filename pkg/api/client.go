package api

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Client provides a high-level interface for interacting with the key-value store
type Client struct {
	mu sync.RWMutex

	// Client ID for session tracking
	clientID string

	// Sequence number for requests
	sequence uint64

	// Server connections (simplified for now)
	servers []*Server

	// Current known leader index
	leaderIndex int

	// Retry configuration
	maxRetries     int
	retryDelay     time.Duration
	requestTimeout time.Duration

	// Random source for generating IDs
	rand *rand.Rand
}

// ClientConfig holds client configuration
type ClientConfig struct {
	MaxRetries     int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		MaxRetries:     3,
		RetryDelay:     100 * time.Millisecond,
		RequestTimeout: 5 * time.Second,
	}
}

// NewClient creates a new client
func NewClient(clientID string, servers []*Server, config *ClientConfig) *Client {
	if config == nil {
		config = DefaultClientConfig()
	}

	return &Client{
		clientID:       clientID,
		sequence:       0,
		servers:        servers,
		leaderIndex:    -1,
		maxRetries:     config.MaxRetries,
		retryDelay:     config.RetryDelay,
		requestTimeout: config.RequestTimeout,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Get retrieves a value from the store
func (c *Client) Get(key string) ([]byte, error) {
	req := c.newRequest(OperationGet, key, nil)

	resp, err := c.executeWithRetry(req)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		if resp.Error == ErrKeyNotFound {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("get failed: %s", resp.Error)
	}

	return resp.Value, nil
}

// Put sets a key-value pair
func (c *Client) Put(key string, value []byte) error {
	req := c.newRequest(OperationPut, key, value)

	resp, err := c.executeWithRetry(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("put failed: %s", resp.Error)
	}

	return nil
}

// Delete removes a key
func (c *Client) Delete(key string) error {
	req := c.newRequest(OperationDelete, key, nil)

	resp, err := c.executeWithRetry(req)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("delete failed: %s", resp.Error)
	}

	return nil
}

// Batch executes a batch of operations
func (c *Client) Batch(operations []Operation) (*BatchResponse, error) {
	batchReq := c.newBatchRequest(operations)

	return c.executeBatchWithRetry(batchReq)
}

// newRequest creates a new request with incremented sequence number
func (c *Client) newRequest(opType OperationType, key string, value []byte) *Request {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sequence++

	return &Request{
		ID:       c.generateRequestID(),
		ClientID: c.clientID,
		Sequence: c.sequence,
		Type:     opType,
		Key:      key,
		Value:    value,
		Timeout:  c.requestTimeout,
	}
}

// newBatchRequest creates a new batch request
func (c *Client) newBatchRequest(operations []Operation) *BatchRequest {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sequence++

	return &BatchRequest{
		ID:         c.generateRequestID(),
		ClientID:   c.clientID,
		Sequence:   c.sequence,
		Operations: operations,
		Timeout:    c.requestTimeout,
	}
}

// executeWithRetry executes a request with retry logic
func (c *Client) executeWithRetry(req *Request) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(delay)
		}

		// Try current leader first
		resp, err := c.execute(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Handle redirection
		if !resp.Success && resp.Error == ErrNotLeader {
			// Update leader and retry
			c.updateLeader(resp.LeaderID)
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d retries: %v", c.maxRetries, lastErr)
	}

	return nil, fmt.Errorf("request failed after %d retries", c.maxRetries)
}

// executeBatchWithRetry executes a batch request with retry logic
func (c *Client) executeBatchWithRetry(req *BatchRequest) (*BatchResponse, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryDelay * time.Duration(1<<uint(attempt-1))
			time.Sleep(delay)
		}

		resp, err := c.executeBatch(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Handle redirection
		if !resp.Success && resp.Error == ErrNotLeader {
			c.updateLeader(resp.LeaderID)
			continue
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("batch request failed after %d retries: %v", c.maxRetries, lastErr)
	}

	return nil, fmt.Errorf("batch request failed after %d retries", c.maxRetries)
}

// execute sends a request to the current server
func (c *Client) execute(req *Request) (*Response, error) {
	server := c.selectServer()
	if server == nil {
		return nil, fmt.Errorf("no servers available")
	}

	return server.HandleRequest(req), nil
}

// executeBatch sends a batch request to the current server
func (c *Client) executeBatch(req *BatchRequest) (*BatchResponse, error) {
	server := c.selectServer()
	if server == nil {
		return nil, fmt.Errorf("no servers available")
	}

	return server.HandleBatch(req), nil
}

// selectServer selects a server to send the request to
func (c *Client) selectServer() *Server {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.servers) == 0 {
		return nil
	}

	// Try known leader first
	if c.leaderIndex >= 0 && c.leaderIndex < len(c.servers) {
		return c.servers[c.leaderIndex]
	}

	// Try a random server
	idx := c.rand.Intn(len(c.servers))
	return c.servers[idx]
}

// updateLeader updates the known leader
func (c *Client) updateLeader(leaderID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// In a real implementation, we would map leaderID to server index
	// For now, we just invalidate the current leader
	c.leaderIndex = -1
}

// generateRequestID generates a unique request ID
func (c *Client) generateRequestID() string {
	return fmt.Sprintf("%s-%d-%d", c.clientID, time.Now().UnixNano(), c.rand.Int63())
}

// SetRequestTimeout sets the request timeout
func (c *Client) SetRequestTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestTimeout = timeout
}

// SetRetryConfig sets the retry configuration
func (c *Client) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxRetries = maxRetries
	c.retryDelay = retryDelay
}

// GetSequence returns the current sequence number
func (c *Client) GetSequence() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sequence
}
