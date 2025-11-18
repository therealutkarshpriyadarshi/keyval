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

Data Commands:
  put <key> <value>                Set a key-value pair
  get <key>                        Get a value by key
  delete <key>                     Delete a key
  scan [prefix]                    Scan keys with optional prefix

Cluster Commands:
  cluster info                     Show cluster information
  cluster health                   Show cluster health
  cluster leader                   Show current leader
  cluster members                  List all cluster members
  cluster member <id>              Show specific member info
  cluster add <id> <address>       Add a new member
  cluster remove <id>              Remove a member
  cluster promote <id>             Promote learner to voter
  cluster replace <old> <new> <addr>  Replace a node
  cluster transfer <id>            Transfer leadership

Maintenance Commands:
  snapshot create                  Create a snapshot
  snapshot list                    List snapshots
  compact                          Compact logs
  backup <path>                    Backup cluster data
  restore <path>                   Restore from backup

Options:
  -server <address>     Server address (default: localhost:7000)
  -timeout <duration>   Request timeout (default: 5s)
  -json                 Output in JSON format
  -h, -help             Show this help message

Examples:
  kvctl put mykey myvalue
  kvctl get mykey
  kvctl cluster members
  kvctl cluster add node4 localhost:7004
  kvctl snapshot create
`
)

var (
	serverAddr = flag.String("server", "localhost:7000", "Server address")
	timeout    = flag.Duration("timeout", 5*time.Second, "Request timeout")
	jsonOutput = flag.Bool("json", false, "Output in JSON format")
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

	clusterCmds := NewClusterCommands(client)
	maintenanceCmds := NewMaintenanceCommands(client)

	var err error
	switch command {
	// Data commands
	case "put":
		err = client.Put(args)
	case "get":
		err = client.Get(args)
	case "delete":
		err = client.Delete(args)
	case "scan":
		err = client.Scan(args)

	// Cluster commands
	case "cluster":
		err = handleClusterCommand(clusterCmds, args)

	// Maintenance commands
	case "snapshot":
		err = handleSnapshotCommand(maintenanceCmds, args)
	case "compact":
		err = maintenanceCmds.Compact()
	case "backup":
		err = handleBackupCommand(maintenanceCmds, args)
	case "restore":
		err = handleRestoreCommand(maintenanceCmds, args)

	// Legacy status command
	case "status":
		err = clusterCmds.ShowClusterInfo(*jsonOutput)

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

func handleClusterCommand(cc *ClusterCommands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("cluster subcommand required (info, health, leader, members, add, remove, etc.)")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "info":
		return cc.ShowClusterInfo(*jsonOutput)
	case "health":
		return cc.ShowClusterHealth(*jsonOutput)
	case "leader":
		return cc.ShowLeader(*jsonOutput)
	case "members":
		return cc.ListMembers(*jsonOutput)
	case "member":
		if len(subargs) != 1 {
			return fmt.Errorf("usage: kvctl cluster member <id>")
		}
		return cc.ShowMember(subargs[0], *jsonOutput)
	case "add":
		if len(subargs) != 2 {
			return fmt.Errorf("usage: kvctl cluster add <id> <address>")
		}
		return cc.AddMember(subargs[0], subargs[1])
	case "remove":
		if len(subargs) != 1 {
			return fmt.Errorf("usage: kvctl cluster remove <id>")
		}
		return cc.RemoveMember(subargs[0], false)
	case "promote":
		if len(subargs) != 1 {
			return fmt.Errorf("usage: kvctl cluster promote <id>")
		}
		return cc.PromoteMember(subargs[0])
	case "transfer":
		if len(subargs) != 1 {
			return fmt.Errorf("usage: kvctl cluster transfer <id>")
		}
		return cc.TransferLeadership(subargs[0], *timeout)
	case "replace":
		if len(subargs) != 3 {
			return fmt.Errorf("usage: kvctl cluster replace <old-id> <new-id> <new-address>")
		}
		return cc.ReplaceNode(subargs[0], subargs[1], subargs[2])
	default:
		return fmt.Errorf("unknown cluster subcommand: %s", subcommand)
	}
}

func handleSnapshotCommand(mc *MaintenanceCommands, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("snapshot subcommand required (create or list)")
	}

	subcommand := args[0]

	switch subcommand {
	case "create":
		return mc.CreateSnapshot()
	case "list":
		return mc.ListSnapshots(*jsonOutput)
	default:
		return fmt.Errorf("unknown snapshot subcommand: %s", subcommand)
	}
}

func handleBackupCommand(mc *MaintenanceCommands, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kvctl backup <path>")
	}
	return mc.Backup(args[0])
}

func handleRestoreCommand(mc *MaintenanceCommands, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: kvctl restore <path>")
	}
	return mc.Restore(args[0])
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

// Scan scans keys with optional prefix
func (c *Client) Scan(args []string) error {
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}

	if prefix == "" {
		fmt.Printf("SCAN all keys (request would be sent to %s)\n", c.serverAddr)
	} else {
		fmt.Printf("SCAN keys with prefix '%s' (request would be sent to %s)\n", prefix, c.serverAddr)
	}

	// In production, this would scan via gRPC
	fmt.Println("\nResults (placeholder):")
	fmt.Println("  key1 = value1")
	fmt.Println("  key2 = value2")
	fmt.Println("  key3 = value3")
	fmt.Println("\nTotal: 3 keys")

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
