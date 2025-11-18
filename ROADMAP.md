# KeyVal: Production-Grade Distributed Database with Raft Consensus

## 🎯 Project Overview
Build a distributed key-value database implementing the full Raft consensus algorithm for strong consistency across cluster nodes.

**Target Capabilities:**
- 10,000+ writes/second (etcd-level performance)
- Linearizable reads and writes
- Automatic leader election and failover
- Network partition tolerance
- Dynamic cluster membership
- Production-ready monitoring and observability

---

## 📋 Development Roadmap

### **Phase 1: Project Foundation & Setup** (Week 1)
**Goal:** Establish project structure and development environment

#### Tasks:
- [x] Initialize Go module and project structure
- [ ] Set up directory structure:
  ```
  keyval/
  ├── cmd/
  │   ├── keyval/          # Main server binary
  │   └── kvctl/           # CLI client tool
  ├── pkg/
  │   ├── raft/            # Core Raft implementation
  │   ├── storage/         # Persistence layer (BoltDB/RocksDB)
  │   ├── rpc/             # gRPC service definitions
  │   ├── statemachine/    # Key-value state machine
  │   ├── cluster/         # Cluster membership management
  │   └── api/             # Client API
  ├── proto/               # Protocol buffer definitions
  ├── test/
  │   ├── integration/     # Integration tests
  │   └── chaos/           # Jepsen-style chaos tests
  ├── docs/                # Documentation
  └── scripts/             # Build and deployment scripts
  ```
- [ ] Add dependencies:
  - gRPC and protobuf
  - BoltDB (`go.etcd.io/bbolt`)
  - Prometheus client
  - Zap logger
  - Testing frameworks
- [ ] Create Makefile for build/test automation
- [ ] Set up CI/CD pipeline (GitHub Actions)
- [ ] Write initial documentation structure

**Deliverables:**
- Compiling project skeleton
- Working build system
- Development environment documentation

---

### **Phase 2: Core Raft - Leader Election** (Week 2-3)
**Goal:** Implement Raft leader election with proper timeout handling

#### Key Concepts:
- Server states: Follower, Candidate, Leader
- Election timeout randomization (150-300ms)
- Term numbers and vote management
- RequestVote RPC

#### Tasks:
- [ ] Define Raft node structure and state
- [ ] Implement persistent state (currentTerm, votedFor, log[])
- [ ] Implement volatile state (commitIndex, lastApplied)
- [ ] Create election timer with randomization
- [ ] Implement RequestVote RPC handler
- [ ] Implement candidate voting logic
- [ ] Handle split votes and re-election
- [ ] Add unit tests for election scenarios:
  - Single node election
  - Multi-node election
  - Network partition during election
  - Concurrent candidates

**Validation:**
- 3-node cluster elects leader within 1 second
- Leader re-election on failure < 2 seconds
- No split-brain scenarios

**Files to Create:**
- `pkg/raft/node.go` - Core node structure
- `pkg/raft/election.go` - Election logic
- `pkg/raft/rpc.go` - RPC definitions
- `proto/raft.proto` - Protocol buffer definitions

---

### **Phase 3: Core Raft - Log Replication** (Week 3-4)
**Goal:** Implement log replication with AppendEntries RPC

#### Key Concepts:
- Log structure and indexing
- AppendEntries RPC (heartbeats and log entries)
- Log matching property
- Commit and apply logic
- Handling follower inconsistencies

#### Tasks:
- [ ] Define log entry structure
- [ ] Implement in-memory log storage
- [ ] Create heartbeat mechanism (50ms intervals)
- [ ] Implement AppendEntries RPC handler
- [ ] Add log consistency checking
- [ ] Implement log replication to followers
- [ ] Add commit index advancement logic
- [ ] Implement apply to state machine
- [ ] Handle log conflicts and repair
- [ ] Add unit tests:
  - Normal operation replication
  - Follower crash recovery
  - Network delays
  - Log conflict resolution

**Validation:**
- All followers replicate leader's log
- Committed entries never lost
- Log consistency maintained across network partitions

**Files to Create:**
- `pkg/raft/log.go` - Log management
- `pkg/raft/replication.go` - Replication logic
- `pkg/raft/append_entries.go` - AppendEntries handler

---

### **Phase 4: Persistence Layer** (Week 5)
**Goal:** Implement durable storage with write-ahead logging

#### Key Concepts:
- Write-ahead log (WAL)
- Persistent state (term, vote, log)
- Crash recovery
- Fsync for durability

#### Tasks:
- [ ] Integrate BoltDB for persistent storage
- [ ] Implement WAL structure:
  - Log entries
  - Metadata (term, votedFor)
  - Snapshot metadata
- [ ] Add persistence hooks in Raft:
  - Persist before responding to RequestVote
  - Persist before adding log entries
- [ ] Implement crash recovery logic
- [ ] Add checksum/CRC for data integrity
- [ ] Implement background log sync
- [ ] Add tests:
  - Crash and recovery
  - Disk full scenarios
  - Corruption detection

**Validation:**
- Node recovers correct state after crash
- No data loss after fsync
- Performance: < 1ms write latency for persistence

**Files to Create:**
- `pkg/storage/wal.go` - Write-ahead log
- `pkg/storage/store.go` - BoltDB interface
- `pkg/raft/persistence.go` - Persistence integration

---

### **Phase 5: Client Operations (Read/Write)** (Week 6)
**Goal:** Implement linearizable client operations

#### Key Concepts:
- Client API (Get, Put, Delete)
- Request routing to leader
- Linearizable reads
- Client request IDs for idempotency
- Session management

#### Tasks:
- [ ] Define key-value state machine interface
- [ ] Implement basic KV operations (Get, Put, Delete)
- [ ] Create client request structure with unique IDs
- [ ] Implement leader-only writes
- [ ] Add request forwarding to leader
- [ ] Implement read-after-write consistency
- [ ] Add linearizable read mechanisms:
  - ReadIndex optimization
  - Lease-based reads (optional)
- [ ] Handle duplicate detection
- [ ] Add client API tests:
  - Basic CRUD operations
  - Linearizability verification
  - Leader failure during request

**Validation:**
- All writes go through Raft log
- Linearizable read guarantees
- Automatic retry on leader failure

**Files to Create:**
- `pkg/statemachine/kv.go` - KV state machine
- `pkg/api/client.go` - Client API
- `pkg/api/server.go` - API server
- `proto/api.proto` - Client API protocol

---

### **Phase 6: Snapshot & Log Compaction** (Week 7)
**Goal:** Implement snapshots to prevent unbounded log growth

#### Key Concepts:
- Snapshot creation
- Snapshot installation (InstallSnapshot RPC)
- Log compaction/truncation
- Incremental snapshots

#### Tasks:
- [ ] Define snapshot format
- [ ] Implement snapshot creation logic
- [ ] Add snapshot trigger policies:
  - Log size threshold
  - Periodic snapshots
- [ ] Implement InstallSnapshot RPC
- [ ] Add snapshot transfer for slow followers
- [ ] Implement log truncation after snapshot
- [ ] Add snapshot metadata tracking
- [ ] Implement snapshot restoration on startup
- [ ] Add tests:
  - Snapshot creation and restoration
  - InstallSnapshot RPC
  - Log compaction correctness

**Validation:**
- Log size remains bounded
- Fast follower catch-up
- No data loss during compaction

**Files to Create:**
- `pkg/raft/snapshot.go` - Snapshot logic
- `pkg/raft/install_snapshot.go` - InstallSnapshot handler
- `pkg/storage/snapshot.go` - Snapshot storage

---

### **Phase 7: Dynamic Cluster Membership** (Week 8-9)
**Goal:** Support adding/removing nodes without downtime

#### Key Concepts:
- Joint consensus (C-old,new)
- Single-server membership changes
- Configuration log entries
- Non-voting members (learners)

#### Tasks:
- [ ] Design configuration change log entries
- [ ] Implement single-server changes (safer approach)
- [ ] Add node addition:
  - New node starts as learner
  - Catches up via snapshots
  - Promoted to voting member
- [ ] Add node removal
- [ ] Implement configuration queries
- [ ] Add safety checks (prevent removing leader)
- [ ] Add tests:
  - Adding nodes
  - Removing nodes
  - Multiple concurrent changes
  - Leader removal

**Validation:**
- No downtime during membership changes
- Cluster maintains quorum
- Safe leader transitions

**Files to Create:**
- `pkg/cluster/membership.go` - Membership management
- `pkg/raft/config_change.go` - Configuration changes
- `cmd/kvctl/membership.go` - CLI commands

---

### **Phase 8: Production Features** (Week 10-11)
**Goal:** Add observability, monitoring, and operational tools

#### Tasks:

**8.1 Monitoring & Metrics:**
- [ ] Integrate Prometheus client
- [ ] Export key metrics:
  - Leader election count
  - Log replication lag
  - Commit latency (p50, p95, p99)
  - Snapshot frequency
  - Request throughput
  - Error rates
- [ ] Create Grafana dashboards
- [ ] Add health check endpoints

**8.2 Distributed Tracing:**
- [ ] Integrate OpenTelemetry
- [ ] Trace request paths
- [ ] Add span annotations for key operations
- [ ] Export to Jaeger/Zipkin

**8.3 Operational Tools:**
- [ ] Create kvctl CLI:
  - Cluster status
  - Member list
  - Add/remove nodes
  - Trigger snapshots
  - Log compaction
- [ ] Add admin API endpoints
- [ ] Create Docker images
- [ ] Write Kubernetes manifests
- [ ] Add graceful shutdown

**8.4 Performance Optimization:**
- [ ] Add pipeline/batching for log replication
- [ ] Implement log entry caching
- [ ] Optimize RPC serialization
- [ ] Add connection pooling
- [ ] Profile and optimize hot paths

**Files to Create:**
- `pkg/metrics/prometheus.go` - Metrics
- `pkg/tracing/opentelemetry.go` - Tracing
- `cmd/kvctl/` - CLI tool
- `deployments/kubernetes/` - K8s manifests

---

### **Phase 9: Testing & Validation** (Week 12-13)
**Goal:** Comprehensive testing including chaos engineering

#### Tasks:

**9.1 Unit Tests:**
- [ ] Achieve >80% code coverage
- [ ] Test all Raft state transitions
- [ ] Test edge cases

**9.2 Integration Tests:**
- [ ] Multi-node cluster tests
- [ ] Leader failure scenarios
- [ ] Network partition tests
- [ ] Slow follower tests
- [ ] Disk failure simulation

**9.3 Chaos Testing (Jepsen-style):**
- [ ] Set up Jepsen test environment
- [ ] Implement linearizability checker
- [ ] Test scenarios:
  - Random node crashes
  - Network partitions
  - Clock skew
  - Packet loss/delays
  - Disk full
- [ ] Generate consistency reports

**9.4 Performance Benchmarks:**
- [ ] Write throughput benchmarks
- [ ] Read throughput benchmarks
- [ ] Latency measurements
- [ ] Scalability tests (3, 5, 7 nodes)
- [ ] Compare with etcd/Consul

**9.5 Correctness Verification:**
- [ ] TLA+ specification (optional)
- [ ] Model checking with TLC
- [ ] Formal verification of key invariants

**Validation Targets:**
- 10,000+ writes/second (3-node cluster)
- <10ms p99 write latency
- Linearizability verification passes
- Zero data loss in chaos tests

**Files to Create:**
- `test/integration/` - Integration tests
- `test/chaos/` - Chaos tests
- `test/bench/` - Benchmarks
- `tla/` - TLA+ specs (optional)

---

## 🏗️ Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        Client Layer                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│  │  kvctl   │  │ gRPC API │  │HTTP API  │                   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘                   │
└───────┼─────────────┼─────────────┼─────────────────────────┘
        │             │             │
        └─────────────┼─────────────┘
                      │
        ┌─────────────▼──────────────┐
        │     API Server Layer       │
        │  - Request validation      │
        │  - Leader forwarding       │
        │  - Session management      │
        └─────────────┬──────────────┘
                      │
        ┌─────────────▼──────────────┐
        │    Raft Consensus Layer    │
        │  ┌──────────────────────┐  │
        │  │   Leader Election    │  │
        │  │  - Term management   │  │
        │  │  - Vote collection   │  │
        │  └──────────────────────┘  │
        │  ┌──────────────────────┐  │
        │  │   Log Replication    │  │
        │  │  - AppendEntries     │  │
        │  │  - Commit tracking   │  │
        │  └──────────────────────┘  │
        │  ┌──────────────────────┐  │
        │  │    Snapshot Mgmt     │  │
        │  │  - Creation          │  │
        │  │  - Installation      │  │
        │  └──────────────────────┘  │
        └─────────────┬──────────────┘
                      │
        ┌─────────────▼──────────────┐
        │   State Machine Layer      │
        │  - Key-Value operations    │
        │  - Command application     │
        │  - Snapshot generation     │
        └─────────────┬──────────────┘
                      │
        ┌─────────────▼──────────────┐
        │    Storage Layer           │
        │  ┌──────────────────────┐  │
        │  │   Write-Ahead Log    │  │
        │  │    (BoltDB/RocksDB)  │  │
        │  └──────────────────────┘  │
        │  ┌──────────────────────┐  │
        │  │   Snapshot Storage   │  │
        │  └──────────────────────┘  │
        └────────────────────────────┘

         Inter-Node Communication
              (gRPC/TCP)
        ┌──────┐  ┌──────┐  ┌──────┐
        │Node 1│  │Node 2│  │Node 3│
        └──────┘  └──────┘  └──────┘
```

---

## 📚 Learning Resources

### Essential Reading:
1. **Raft Paper:** "In Search of an Understandable Consensus Algorithm" by Ongaro & Ousterhout
2. **Raft Visualization:** https://raft.github.io/
3. **etcd Documentation:** https://etcd.io/docs/
4. **Designing Data-Intensive Applications** by Martin Kleppmann (Chapter 9: Consistency and Consensus)

### Implementation References:
- etcd/raft: https://github.com/etcd-io/raft
- Hashicorp Raft: https://github.com/hashicorp/raft
- Consul: https://github.com/hashicorp/consul

### Testing:
- Jepsen: https://github.com/jepsen-io/jepsen
- Kyle Kingsbury's Jepsen analyses: https://jepsen.io/analyses

---

## 🎯 Success Metrics

### Functional:
- ✅ Leader election converges in <2s
- ✅ Log replication maintains consistency
- ✅ Linearizable read/write operations
- ✅ Survives network partitions
- ✅ Dynamic membership changes work

### Performance:
- ✅ 10,000+ writes/second (3-node cluster)
- ✅ <5ms p50 write latency
- ✅ <10ms p99 write latency
- ✅ <1ms read latency (lease-based)

### Operational:
- ✅ Zero data loss in chaos tests
- ✅ Automatic failover <2s
- ✅ Grafana dashboards functional
- ✅ Distributed tracing integrated
- ✅ Docker + Kubernetes deployment ready

---

## 🚀 Getting Started

Start with Phase 1 - let's build the foundation!

```bash
# Initialize the project
make init

# Run tests
make test

# Build binaries
make build

# Start a 3-node cluster locally
make cluster-start

# Run benchmarks
make bench
```

---

## 📝 Notes

- **Complexity:** This is a 3-4 month project for production quality
- **Learning curve:** Expect to spend significant time understanding Raft
- **Testing:** Allocate 30-40% of time to testing and validation
- **Performance:** Optimization should come after correctness

**Focus areas for interviews:**
- Explain CAP theorem tradeoffs
- Describe leader election process
- Explain log replication guarantees
- Discuss failure scenarios and recovery
- Performance optimization strategies
