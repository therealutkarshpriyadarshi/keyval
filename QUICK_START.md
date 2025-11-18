# Quick Start Guide

## Prerequisites

- Go 1.21 or later
- Docker (for testing multi-node clusters)
- Make
- Protocol Buffers compiler (protoc)

## Initial Setup

### 1. Initialize the Go Module

```bash
go mod init github.com/therealutkarshpriyadarshi/keyval
```

### 2. Install Development Tools

```bash
# Install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install testing tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### 3. Install Core Dependencies

```bash
# gRPC and Protocol Buffers
go get -u google.golang.org/grpc
go get -u google.golang.org/protobuf

# BoltDB for persistence
go get -u go.etcd.io/bbolt

# Logging
go get -u go.uber.org/zap

# Prometheus metrics
go get -u github.com/prometheus/client_golang

# Testing
go get -u github.com/stretchr/testify
```

## Project Structure Setup

Run this to create the directory structure:

```bash
mkdir -p cmd/{keyval,kvctl}
mkdir -p pkg/{raft,storage,rpc,statemachine,cluster,api}
mkdir -p proto
mkdir -p test/{integration,chaos}
mkdir -p docs
mkdir -p scripts
mkdir -p deployments/{docker,kubernetes}
```

## Development Workflow

### Phase 1: Foundation (Start Here!)

1. **Week 1, Day 1-2: Project Setup**
   - ✅ Initialize Go module
   - ✅ Set up directory structure
   - Create Makefile
   - Add .gitignore
   - Set up CI/CD

2. **Week 1, Day 3-4: Core Types & Interfaces**
   - Define Raft node structure
   - Create protocol buffer definitions
   - Set up logging infrastructure
   - Add configuration management

3. **Week 1, Day 5-7: Testing Infrastructure**
   - Set up test framework
   - Create mock implementations
   - Add test utilities
   - Write first tests

### Running Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/raft/...

# Run with coverage
make coverage

# Run integration tests
make integration-test
```

### Building

```bash
# Build all binaries
make build

# Build server only
make build-server

# Build CLI only
make build-cli
```

## First Implementation Goals

### Milestone 1: Single Node Raft (Week 2)
- Node can become leader
- Node maintains term
- Basic logging works
- State persists to disk

**Test:**
```bash
./bin/keyval --node-id=1 --cluster=node1
# Should elect itself as leader
```

### Milestone 2: Multi-Node Election (Week 2-3)
- 3 nodes can elect a leader
- Leader sends heartbeats
- Followers respond to heartbeats
- Re-election works on leader failure

**Test:**
```bash
# Terminal 1
./bin/keyval --node-id=1 --cluster=node1,node2,node3 --port=8001

# Terminal 2
./bin/keyval --node-id=2 --cluster=node1,node2,node3 --port=8002

# Terminal 3
./bin/keyval --node-id=3 --cluster=node1,node2,node3 --port=8003

# Check leader
./bin/kvctl status
```

### Milestone 3: Log Replication (Week 3-4)
- Leader accepts write requests
- Leader replicates to followers
- Commits happen after majority
- Followers apply committed entries

**Test:**
```bash
# Write a key
./bin/kvctl put mykey myvalue

# Read from different nodes
./bin/kvctl get mykey --node=node1
./bin/kvctl get mykey --node=node2
./bin/kvctl get mykey --node=node3
```

## Debugging Tips

### Enable Debug Logging

```bash
./bin/keyval --log-level=debug
```

### Monitor Raft State

```bash
# Watch cluster status
watch -n 1 './bin/kvctl status'

# Check node metrics
curl http://localhost:9001/metrics
```

### Common Issues

**Issue: Nodes can't reach each other**
- Check firewall settings
- Verify port configuration
- Check network connectivity

**Issue: Elections keep timing out**
- Check election timeout configuration
- Verify heartbeat interval
- Look for network delays

**Issue: Log replication stuck**
- Check follower logs
- Verify AppendEntries RPC
- Check for disk I/O issues

## Performance Testing

### Basic Throughput Test

```bash
# Run benchmark
go test -bench=BenchmarkWrite -benchtime=10s ./test/bench/

# Expected results (target):
# BenchmarkWrite-8    100000    5000 ns/op    10000 ops/sec
```

### Latency Test

```bash
# Measure p50, p95, p99 latencies
./bin/kvctl benchmark --duration=60s --rate=1000
```

## Next Steps

1. **Read the Raft Paper** (2-3 days)
   - https://raft.github.io/raft.pdf
   - Focus on sections 5-6 (core algorithm)
   - Use https://raft.github.io/ for visualization

2. **Study etcd/raft Implementation** (2-3 days)
   - Clone: https://github.com/etcd-io/raft
   - Read the code structure
   - Understand their design decisions

3. **Start Coding!** (Week 2+)
   - Begin with Phase 2: Leader Election
   - Write tests first (TDD approach)
   - Commit frequently
   - Document as you go

## Resources for Each Phase

### Phase 2: Leader Election
- Raft paper Section 5.1-5.2
- etcd/raft: `raft.go`, `node.go`
- Key files to study:
  - How election timer works
  - Vote granting logic
  - Term management

### Phase 3: Log Replication
- Raft paper Section 5.3-5.4
- etcd/raft: `log.go`, `raft.go` (replication)
- Key concepts:
  - Log matching property
  - Handling inconsistencies
  - Commit rules

### Phase 4: Persistence
- Raft paper Section 5.4.1
- etcd/wal: https://github.com/etcd-io/etcd/tree/main/server/wal
- Key topics:
  - fsync guarantees
  - Recovery procedures
  - Corruption handling

## Interview Preparation

As you build, prepare to answer:

1. **Why Raft over Paxos?**
   - Understandability
   - Strong leader
   - Simpler protocol

2. **What happens during a network partition?**
   - Minority partition can't commit
   - Majority partition continues
   - How to handle split-brain

3. **How do you ensure linearizability?**
   - All writes through log
   - ReadIndex for consistent reads
   - Client request tracking

4. **Performance optimizations?**
   - Pipeline/batching
   - Lease-based reads
   - Parallel log application

5. **CAP theorem tradeoffs?**
   - Raft chooses CP (Consistency + Partition tolerance)
   - Availability impacted during partitions
   - Why this is acceptable for coordination systems

## Contribution Guidelines

Once you have a working implementation:

1. Write comprehensive tests
2. Document all public APIs
3. Add examples
4. Create benchmark comparisons
5. Write blog posts about your learnings

## Getting Help

- **Raft Google Group:** https://groups.google.com/g/raft-dev
- **etcd Discussion:** https://github.com/etcd-io/etcd/discussions
- **DDIA Book Club:** Various online communities
- **System Design interviews:** Practice explaining your implementation

---

**Remember:** Building a production-grade distributed database is a marathon, not a sprint. Take it phase by phase, understand each component deeply, and test thoroughly. This project will teach you more about distributed systems than any course or book!

Good luck! 🚀
