package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/therealutkarshpriyadarshi/keyval/pkg/api"
	"github.com/therealutkarshpriyadarshi/keyval/pkg/raft"
)

var (
	nodeID      = flag.String("id", "", "Node ID (required)")
	addr        = flag.String("addr", "127.0.0.1:7000", "Node address")
	peers       = flag.String("peers", "", "Comma-separated list of peer addresses (id=addr,id=addr)")
	logLevel    = flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
	dataDir     = flag.String("data-dir", "./data", "Data directory for persistent storage")
	showVersion = flag.Bool("version", false, "Show version information")
)

const version = "0.1.0"

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("KeyVal version %s\n", version)
		os.Exit(0)
	}

	if *nodeID == "" {
		fmt.Fprintf(os.Stderr, "Error: -id flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Configure logger
	logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", *nodeID), log.LstdFlags)

	// Parse peers
	peerList, err := parsePeers(*peers)
	if err != nil {
		logger.Fatalf("Failed to parse peers: %v", err)
	}

	logger.Printf("Starting KeyVal node %s at %s", *nodeID, *addr)
	logger.Printf("Peers: %v", peerList)

	// Create Raft node
	node, err := raft.NewNode(*nodeID, *addr, peerList, raft.WithLogger(logger))
	if err != nil {
		logger.Fatalf("Failed to create Raft node: %v", err)
	}

	// Start Raft node
	if err := node.Start(); err != nil {
		logger.Fatalf("Failed to start Raft node: %v", err)
	}

	// Create API server
	apiServer := api.NewServer(node)
	logger.Printf("API server initialized")

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Printf("KeyVal node %s is running", *nodeID)
	logger.Printf("Press Ctrl+C to stop")

	// Wait for shutdown signal
	sig := <-sigCh
	logger.Printf("Received signal %v, shutting down...", sig)

	// Graceful shutdown
	if err := node.Stop(); err != nil {
		logger.Printf("Error stopping node: %v", err)
	}

	// Clean up API server (in production, this would stop the gRPC server)
	_ = apiServer

	logger.Printf("KeyVal node %s stopped", *nodeID)
}

// parsePeers parses the peer list from command-line argument
// Format: "id1=addr1,id2=addr2,id3=addr3"
func parsePeers(peersStr string) ([]raft.Peer, error) {
	if peersStr == "" {
		return []raft.Peer{}, nil
	}

	parts := strings.Split(peersStr, ",")
	peers := make([]raft.Peer, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid peer format: %s (expected id=addr)", part)
		}

		peers = append(peers, raft.Peer{
			ID:      strings.TrimSpace(kv[0]),
			Address: strings.TrimSpace(kv[1]),
		})
	}

	return peers, nil
}
