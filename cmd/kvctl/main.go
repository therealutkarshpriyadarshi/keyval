package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
)

const (
	usage = `kvctl - KeyVal CLI Tool

Usage:
  kvctl <command> [arguments]

Commands:
  put <key> <value>     Set a key-value pair
  get <key>             Get a value by key
  delete <key>          Delete a key
  status                Show cluster status

Options:
  -server <address>     Server address (default: localhost:7000)
  -timeout <duration>   Request timeout (default: 5s)
  -h, -help             Show this help message

Examples:
  kvctl put mykey myvalue
  kvctl get mykey
  kvctl delete mykey
  kvctl -server localhost:8000 get mykey
`
)

var (
	serverAddr = flag.String("server", "localhost:7000", "Server address")
	timeout    = flag.Duration("timeout", 5*time.Second, "Request timeout")
	help       = flag.Bool("help", false, "Show help message")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}

	flag.Parse()

	if *help || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	command := flag.Arg(0)
	args := flag.Args()[1:]

	// Create client (placeholder - in production this would connect via gRPC)
	client := &Client{
		serverAddr: *serverAddr,
		timeout:    *timeout,
	}

	var err error
	switch command {
	case "put":
		err = client.Put(args)
	case "get":
		err = client.Get(args)
	case "delete":
		err = client.Delete(args)
	case "status":
		err = client.Status()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Client represents a kvctl client
type Client struct {
	serverAddr string
	timeout    time.Duration
	clientID   string
	sequence   uint64
}

// Put sets a key-value pair
func (c *Client) Put(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: kvctl put <key> <value>")
	}

	key := args[0]
	value := args[1]

	req := &api.Request{
		ID:       generateRequestID(),
		ClientID: c.getClientID(),
		Sequence: c.nextSequence(),
		Type:     api.OperationPut,
		Key:      key,
		Value:    []byte(value),
		Timeout:  c.timeout,
	}

	// In production, this would send the request via gRPC
	// For now, we'll just print what we would do
	fmt.Printf("PUT %s = %s (request would be sent to %s)\n", key, value, c.serverAddr)
	fmt.Printf("Request ID: %s\n", req.ID)
	fmt.Println("Success: true")

	return nil
}

// Get retrieves a value by key
func (c *Client) Get(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kvctl get <key>")
	}

	key := args[0]

	req := &api.Request{
		ID:       generateRequestID(),
		ClientID: c.getClientID(),
		Sequence: c.nextSequence(),
		Type:     api.OperationGet,
		Key:      key,
		Timeout:  c.timeout,
	}

	// In production, this would send the request via gRPC
	fmt.Printf("GET %s (request would be sent to %s)\n", key, c.serverAddr)
	fmt.Printf("Request ID: %s\n", req.ID)
	fmt.Println("(Value would be displayed here)")

	return nil
}

// Delete removes a key
func (c *Client) Delete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kvctl delete <key>")
	}

	key := args[0]

	req := &api.Request{
		ID:       generateRequestID(),
		ClientID: c.getClientID(),
		Sequence: c.nextSequence(),
		Type:     api.OperationDelete,
		Key:      key,
		Timeout:  c.timeout,
	}

	// In production, this would send the request via gRPC
	fmt.Printf("DELETE %s (request would be sent to %s)\n", key, c.serverAddr)
	fmt.Printf("Request ID: %s\n", req.ID)
	fmt.Println("Success: true")

	return nil
}

// Status shows cluster status
func (c *Client) Status() error {
	// In production, this would query the cluster status via gRPC
	fmt.Printf("Cluster Status (from %s):\n", c.serverAddr)
	fmt.Println("  Status: Running (placeholder)")
	fmt.Println("  Leader: node1 (placeholder)")
	fmt.Println("  Nodes: 3/3 (placeholder)")

	return nil
}

// getClientID returns the client ID (generate once per client instance)
func (c *Client) getClientID() string {
	if c.clientID == "" {
		hostname, _ := os.Hostname()
		c.clientID = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}
	return c.clientID
}

// nextSequence returns the next sequence number for this client
func (c *Client) nextSequence() uint64 {
	c.sequence++
	return c.sequence
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

// formatKey formats a key for display
func formatKey(key string) string {
	if len(key) > 50 {
		return key[:47] + "..."
	}
	return key
}

// formatValue formats a value for display
func formatValue(value []byte) string {
	str := string(value)
	if len(str) > 100 {
		str = str[:97] + "..."
	}

	// Replace newlines with \n for display
	str = strings.ReplaceAll(str, "\n", "\\n")
	str = strings.ReplaceAll(str, "\r", "\\r")
	str = strings.ReplaceAll(str, "\t", "\\t")

	return str
}
