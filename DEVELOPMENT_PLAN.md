# KeyVal Development Plan - Detailed Task Breakdown

> A week-by-week, task-by-task implementation guide for building a production-grade distributed database with Raft consensus.

**Project Duration:** 13 weeks
**Target:** Production-ready distributed key-value database
**Current Status:** Phase 1 Complete ✅

---

## Week 1: Project Foundation ✅

### Day 1-2: Repository Setup
- [x] Initialize Git repository
- [x] Create Go module
- [x] Set up directory structure
- [x] Add .gitignore
- [x] Create LICENSE (MIT)
- [x] Write initial README.md

### Day 3-4: Build Infrastructure
- [x] Create comprehensive Makefile
- [x] Set up GitHub Actions CI/CD
- [x] Configure golangci-lint
- [x] Add pre-commit hooks (optional)
- [x] Create CONTRIBUTING.md

### Day 5-7: Documentation
- [x] Write ROADMAP.md
- [x] Create QUICK_START.md
- [x] Add CHANGELOG.md
- [x] Document package structures
- [x] Create architecture diagrams
- [x] Commit and push foundation

**Deliverables:** ✅ Complete project infrastructure and documentation

---

## Week 2: Core Raft Structures & Leader Election (Part 1)

### Day 1: Protocol Buffers & RPC Definitions
- [ ] Create `proto/raft.proto`
  - [ ] Define RequestVote RPC
  - [ ] Define AppendEntries RPC (structure only)
  - [ ] Define InstallSnapshot RPC (structure only)
  - [ ] Define common types (LogEntry, etc.)
- [ ] Generate Go code from protobuf
  - [ ] Run `make proto`
  - [ ] Verify generated files
- [ ] Create `pkg/rpc/client.go` and `pkg/rpc/server.go` stubs

**Files to create:**
```
proto/raft.proto
pkg/rpc/client.go
pkg/rpc/server.go
```

### Day 2: Core Data Structures
- [ ] Create `pkg/raft/types.go`
  - [ ] Define NodeState enum (Follower, Candidate, Leader)
  - [ ] Define LogEntry struct
  - [ ] Define ServerState (persistent)
  - [ ] Define VolatileState
  - [ ] Define LeaderState
- [ ] Create `pkg/raft/config.go`
  - [ ] Election timeout range (150-300ms)
  - [ ] Heartbeat interval (50ms)
  - [ ] Other configuration parameters
- [ ] Add unit tests for data structures

**Files to create:**
```
pkg/raft/types.go
pkg/raft/config.go
pkg/raft/types_test.go
```

### Day 3: Node Structure
- [ ] Create `pkg/raft/node.go`
  - [ ] Define Node struct with all state
  - [ ] Implement NewNode() constructor
  - [ ] Add mutex for thread safety
  - [ ] Implement state getters/setters
  - [ ] Add term management functions
- [ ] Create `pkg/raft/node_test.go`
  - [ ] Test node creation
  - [ ] Test state transitions
  - [ ] Test thread safety

**Files to create:**
```
pkg/raft/node.go
pkg/raft/node_test.go
```

### Day 4: Election Timer
- [ ] Create `pkg/raft/timer.go`
  - [ ] Implement randomized election timeout
  - [ ] Create timer reset mechanism
  - [ ] Add timer stop/start functions
  - [ ] Make it thread-safe
- [ ] Create `pkg/raft/timer_test.go`
  - [ ] Test timeout randomization
  - [ ] Test reset behavior
  - [ ] Test concurrent access

**Files to create:**
```
pkg/raft/timer.go
pkg/raft/timer_test.go
```

### Day 5: RequestVote RPC - Requester Side
- [ ] Create `pkg/raft/election.go`
  - [ ] Implement startElection() function
  - [ ] Increment term
  - [ ] Vote for self
  - [ ] Send RequestVote to all peers
  - [ ] Collect votes
  - [ ] Handle majority wins
- [ ] Add election state management
- [ ] Write unit tests

**Files to create:**
```
pkg/raft/election.go
pkg/raft/election_test.go
```

### Day 6-7: RequestVote RPC - Handler Side & Integration
- [ ] Implement RequestVote handler
  - [ ] Check term comparison
  - [ ] Check votedFor state
  - [ ] Check log up-to-date (prepare for later)
  - [ ] Grant/deny vote
  - [ ] Reset election timer on valid request
- [ ] Integrate with Node
- [ ] Write comprehensive tests
  - [ ] Single node election
  - [ ] Three node election
  - [ ] Five node election
  - [ ] Split vote scenarios
  - [ ] Concurrent candidates

**Files to update:**
```
pkg/raft/election.go
pkg/raft/rpc_handlers.go (new)
pkg/raft/election_test.go
```

**Week 2 Deliverables:**
- RequestVote RPC fully implemented
- Election timer working
- Basic leader election (without log comparison)
- Comprehensive unit tests

---

## Week 3: Leader Election Complete & Log Replication (Part 1)

### Day 1: Complete Log Comparison in Elections
- [ ] Create `pkg/raft/log.go`
  - [ ] Define Log struct
  - [ ] Implement Append() method
  - [ ] Implement Get() method
  - [ ] Implement LastIndex() and LastTerm()
  - [ ] Add isUpToDate() comparison logic
- [ ] Update RequestVote handler to check log
- [ ] Add tests for log-based voting

**Files to create:**
```
pkg/raft/log.go
pkg/raft/log_test.go
```

### Day 2: Heartbeat Mechanism
- [ ] Create `pkg/raft/heartbeat.go`
  - [ ] Implement heartbeat timer (50ms)
  - [ ] Send empty AppendEntries as heartbeat
  - [ ] Handle heartbeat responses
- [ ] Update leader state to track followers
- [ ] Add follower state to process heartbeats
- [ ] Test heartbeat mechanism

**Files to create:**
```
pkg/raft/heartbeat.go
pkg/raft/heartbeat_test.go
```

### Day 3: AppendEntries RPC - Structure
- [ ] Define AppendEntries RPC in protobuf (if not done)
- [ ] Create `pkg/raft/append_entries.go`
  - [ ] Implement AppendEntries request builder
  - [ ] Add consistency check logic
  - [ ] Handle success/failure responses
- [ ] Create handler stub
- [ ] Basic tests

**Files to create:**
```
pkg/raft/append_entries.go
pkg/raft/append_entries_test.go
```

### Day 4: AppendEntries RPC - Handler Implementation
- [ ] Implement AppendEntries handler
  - [ ] Reply false if term < currentTerm
  - [ ] Reply false if log doesn't match prevLogIndex/prevLogTerm
  - [ ] Delete conflicting entries
  - [ ] Append new entries
  - [ ] Update commitIndex
  - [ ] Reset election timer
- [ ] Add comprehensive tests

**Files to update:**
```
pkg/raft/append_entries.go
pkg/raft/rpc_handlers.go
pkg/raft/append_entries_test.go
```

### Day 5: Log Replication - Leader Side
- [ ] Implement log replication logic
  - [ ] Track nextIndex for each follower
  - [ ] Track matchIndex for each follower
  - [ ] Send AppendEntries with entries
  - [ ] Handle success: update nextIndex and matchIndex
  - [ ] Handle failure: decrement nextIndex and retry
- [ ] Add replication tests

**Files to create:**
```
pkg/raft/replication.go
pkg/raft/replication_test.go
```

### Day 6: Commit Index Advancement
- [ ] Implement commit logic
  - [ ] Find highest N where majority has matchIndex ≥ N
  - [ ] Verify log[N].term == currentTerm
  - [ ] Update commitIndex to N
  - [ ] Apply committed entries to state machine (stub)
- [ ] Add commit tests

**Files to update:**
```
pkg/raft/replication.go
pkg/raft/commit.go (new)
pkg/raft/commit_test.go (new)
```

### Day 7: Integration Testing - Leader Election + Replication
- [ ] Create `test/integration/basic_test.go`
  - [ ] Test 3-node cluster formation
  - [ ] Test leader election
  - [ ] Test log replication
  - [ ] Test leader failure and re-election
  - [ ] Test log consistency after election
- [ ] Fix any bugs found

**Files to create:**
```
test/integration/basic_test.go
test/integration/helpers.go
```

**Week 3 Deliverables:**
- Complete leader election with log comparison
- AppendEntries RPC implemented
- Basic log replication working
- Commit index advancement
- Integration tests passing

---

## Week 4: Log Replication Complete & Basic Client Interface

### Day 1: Handle Log Inconsistencies
- [ ] Optimize log backtracking
  - [ ] Implement fast backup (send conflictIndex and conflictTerm)
  - [ ] Leader skips all entries in conflicting term
- [ ] Add tests for various inconsistency scenarios
- [ ] Test follower with extra entries
- [ ] Test follower with missing entries
- [ ] Test follower with conflicting entries

**Files to update:**
```
pkg/raft/append_entries.go
pkg/raft/replication.go
test/integration/log_conflict_test.go (new)
```

### Day 2: Apply to State Machine
- [ ] Create `pkg/statemachine/interface.go`
  - [ ] Define StateMachine interface
  - [ ] Apply(entry) method
  - [ ] Snapshot() method
  - [ ] Restore(snapshot) method
- [ ] Create `pkg/statemachine/kv.go` (basic implementation)
  - [ ] In-memory map
  - [ ] Get/Put/Delete operations
  - [ ] Thread-safe
- [ ] Integrate with Raft node

**Files to create:**
```
pkg/statemachine/interface.go
pkg/statemachine/kv.go
pkg/statemachine/kv_test.go
```

### Day 3: Client Request Handling - Write Path
- [ ] Create `pkg/api/server.go`
  - [ ] Define client request structure
  - [ ] Implement request ID for idempotency
  - [ ] Add request to log (leader only)
  - [ ] Wait for commit
  - [ ] Return result to client
- [ ] Add leader forwarding (redirect to leader)
- [ ] Handle leader changes during request

**Files to create:**
```
pkg/api/server.go
pkg/api/types.go
pkg/api/server_test.go
```

### Day 4: Client Request Handling - Read Path
- [ ] Implement linearizable reads
  - [ ] Option 1: Read through log (safe but slow)
  - [ ] Option 2: ReadIndex optimization
    - [ ] Leader checks if committed entry from current term
    - [ ] Leader sends heartbeats to ensure leadership
    - [ ] Return data once commitIndex ≥ readIndex
- [ ] Add tests for read consistency

**Files to update:**
```
pkg/api/server.go
pkg/raft/read.go (new)
pkg/api/read_test.go (new)
```

### Day 5: Basic CLI Tool
- [ ] Create `cmd/kvctl/main.go`
  - [ ] Command structure (cobra or flag)
  - [ ] Put command
  - [ ] Get command
  - [ ] Delete command
  - [ ] Status command
- [ ] Add connection to server
- [ ] Basic error handling

**Files to create:**
```
cmd/kvctl/main.go
cmd/kvctl/commands.go
```

### Day 6: Server Binary
- [ ] Create `cmd/keyval/main.go`
  - [ ] Parse command-line flags
  - [ ] Initialize Raft node
  - [ ] Start API server
  - [ ] Setup graceful shutdown
  - [ ] Add signal handling
- [ ] Add logging configuration
- [ ] Create basic config file support

**Files to create:**
```
cmd/keyval/main.go
pkg/config/config.go
```

### Day 7: Integration Testing - Full Stack
- [ ] Test full request flow
  - [ ] Client → API → Raft → StateMachine
  - [ ] Write and verify data
  - [ ] Read consistency
  - [ ] Leader failure during request
- [ ] Create test cluster utilities
- [ ] Document test setup

**Files to create:**
```
test/integration/client_test.go
test/integration/cluster.go
```

**Week 4 Deliverables:**
- Log conflict resolution working
- State machine integration complete
- Client API (Get, Put, Delete) functional
- Basic kvctl CLI tool
- Server binary runs
- End-to-end tests passing

---

## Week 5: Persistence Layer

### Day 1: Write-Ahead Log (WAL) - Design
- [ ] Create `pkg/storage/wal.go`
  - [ ] Design WAL entry format
  - [ ] Design segment files
  - [ ] Plan index structure
  - [ ] Define interfaces
- [ ] Add BoltDB dependency
  ```bash
  go get go.etcd.io/bbolt
  ```

**Files to create:**
```
pkg/storage/wal.go
pkg/storage/types.go
```

### Day 2: WAL - Write Operations
- [ ] Implement WAL writing
  - [ ] Append entry to WAL
  - [ ] Implement fsync for durability
  - [ ] Handle segment rotation
  - [ ] Add checksums (CRC32)
- [ ] Add write tests

**Files to update:**
```
pkg/storage/wal.go
pkg/storage/wal_test.go (new)
```

### Day 3: WAL - Read Operations
- [ ] Implement WAL reading
  - [ ] Read entries from WAL
  - [ ] Verify checksums
  - [ ] Handle corruption
  - [ ] Implement replay logic
- [ ] Add read tests

**Files to update:**
```
pkg/storage/wal.go
pkg/storage/wal_test.go
```

### Day 4: Persistent State Storage
- [ ] Create `pkg/storage/store.go`
  - [ ] Initialize BoltDB
  - [ ] Store currentTerm
  - [ ] Store votedFor
  - [ ] Store log entries
  - [ ] Implement batch operations
- [ ] Add persistence tests

**Files to create:**
```
pkg/storage/store.go
pkg/storage/store_test.go
```

### Day 5: Crash Recovery
- [ ] Create `pkg/raft/persistence.go`
  - [ ] Persist before responding to RequestVote
  - [ ] Persist before appending entries
  - [ ] Implement recovery on startup
  - [ ] Restore state from WAL and BoltDB
  - [ ] Validate recovered state
- [ ] Integration with Node

**Files to create:**
```
pkg/raft/persistence.go
pkg/raft/recovery.go
pkg/raft/recovery_test.go
```

### Day 6: Crash Recovery Testing
- [ ] Create crash simulation tests
  - [ ] Kill node mid-operation
  - [ ] Verify state after restart
  - [ ] Test with various crash points
  - [ ] Verify no data loss after fsync
- [ ] Test corruption detection

**Files to create:**
```
test/integration/crash_test.go
```

### Day 7: Performance & Optimization
- [ ] Benchmark persistence layer
  - [ ] Measure write latency
  - [ ] Measure read latency
  - [ ] Profile disk I/O
- [ ] Optimize if needed
  - [ ] Batch writes
  - [ ] Async fsync (optional)
  - [ ] Buffer pool
- [ ] Document performance characteristics

**Files to create:**
```
test/bench/persistence_bench_test.go
```

**Week 5 Deliverables:**
- WAL implementation complete
- BoltDB integration working
- Crash recovery functional
- No data loss guarantees
- Performance benchmarks documented

---

## Week 6: Client Operations Polish & Linearizability

### Day 1: Request Deduplication
- [ ] Implement client session tracking
  - [ ] Assign unique client IDs
  - [ ] Track request sequences
  - [ ] Detect duplicate requests
  - [ ] Return cached results
- [ ] Add session cleanup

**Files to create:**
```
pkg/api/session.go
pkg/api/session_test.go
```

### Day 2: ReadIndex Optimization
- [ ] Implement ReadIndex fully
  - [ ] Track commit index on leader
  - [ ] Send heartbeats before read
  - [ ] Wait for apply
  - [ ] Return result
- [ ] Compare performance: read-through-log vs ReadIndex
- [ ] Add tests

**Files to update:**
```
pkg/raft/read.go
test/bench/read_bench_test.go (new)
```

### Day 3: Lease-Based Reads (Optional but Recommended)
- [ ] Implement read leases
  - [ ] Leader maintains lease
  - [ ] Heartbeat response extends lease
  - [ ] Serve reads from lease
  - [ ] Step down when lease expires
- [ ] Add safety checks
- [ ] Test lease expiration

**Files to create:**
```
pkg/raft/lease.go
pkg/raft/lease_test.go
```

### Day 4: Batch Operations
- [ ] Add batch write support
  - [ ] Accept multiple operations
  - [ ] Single log entry for batch
  - [ ] Atomic application
- [ ] Add batch API endpoints
- [ ] Test atomicity

**Files to update:**
```
pkg/api/server.go
pkg/statemachine/kv.go
test/integration/batch_test.go (new)
```

### Day 5: Linearizability Verification
- [ ] Implement linearizability checker
  - [ ] Record operation history
  - [ ] Use Porcupine or similar library
  - [ ] Verify all reads/writes
- [ ] Create verification tests
- [ ] Run against cluster

**Files to create:**
```
test/integration/linearizability_test.go
test/integration/history.go
```

### Day 6: Error Handling & Retries
- [ ] Improve error handling
  - [ ] Client-side retry logic
  - [ ] Exponential backoff
  - [ ] Leader rediscovery
  - [ ] Timeout handling
- [ ] Add detailed error messages
- [ ] Test error scenarios

**Files to update:**
```
pkg/api/client.go (new)
pkg/api/errors.go (new)
```

### Day 7: End-to-End Testing
- [ ] Create comprehensive E2E tests
  - [ ] Multi-client concurrent writes
  - [ ] Verify linearizability
  - [ ] Leader failures during operations
  - [ ] Network partitions
- [ ] Load testing
- [ ] Document test results

**Files to create:**
```
test/integration/e2e_test.go
test/integration/load_test.go
```

**Week 6 Deliverables:**
- Request deduplication working
- ReadIndex optimization complete
- Lease-based reads (optional)
- Batch operations supported
- Linearizability verified
- Comprehensive error handling

---

## Week 7: Snapshot & Log Compaction

### Day 1: Snapshot Format & Design
- [ ] Design snapshot format
  - [ ] State machine data
  - [ ] Last included index/term
  - [ ] Cluster configuration
  - [ ] Checksum
- [ ] Create `pkg/storage/snapshot.go`
- [ ] Define snapshot metadata

**Files to create:**
```
pkg/storage/snapshot.go
pkg/storage/snapshot_types.go
```

### Day 2: Snapshot Creation
- [ ] Implement snapshot creation
  - [ ] Trigger on log size threshold
  - [ ] Capture state machine state
  - [ ] Write to disk
  - [ ] Update metadata
  - [ ] Make atomic (temp file + rename)
- [ ] Add creation tests

**Files to update:**
```
pkg/raft/snapshot.go (new)
pkg/storage/snapshot.go
test/integration/snapshot_test.go (new)
```

### Day 3: Log Truncation
- [ ] Implement log compaction
  - [ ] Keep entries after snapshot
  - [ ] Truncate old entries
  - [ ] Update log indices
  - [ ] Clean up WAL segments
- [ ] Verify correctness
- [ ] Test edge cases

**Files to update:**
```
pkg/raft/log.go
pkg/storage/wal.go
pkg/raft/snapshot.go
```

### Day 4: Snapshot Restoration
- [ ] Implement snapshot restore
  - [ ] Load on startup
  - [ ] Verify checksum
  - [ ] Restore state machine
  - [ ] Update Raft state
  - [ ] Handle corruption
- [ ] Test recovery from snapshot

**Files to update:**
```
pkg/raft/recovery.go
pkg/storage/snapshot.go
test/integration/snapshot_recovery_test.go (new)
```

### Day 5: InstallSnapshot RPC
- [ ] Implement InstallSnapshot RPC
  - [ ] Send snapshot to follower
  - [ ] Handle large snapshots (chunking)
  - [ ] Verify on receiver
  - [ ] Apply to state machine
  - [ ] Update log
- [ ] Add RPC tests

**Files to create:**
```
pkg/raft/install_snapshot.go
pkg/raft/install_snapshot_test.go
```

### Day 6: Snapshot Transfer
- [ ] Implement leader snapshot sending
  - [ ] Detect when follower is too far behind
  - [ ] Send InstallSnapshot instead of AppendEntries
  - [ ] Resume normal replication after
- [ ] Optimize transfer (compression)
- [ ] Test slow follower catch-up

**Files to update:**
```
pkg/raft/replication.go
pkg/raft/install_snapshot.go
test/integration/slow_follower_test.go (new)
```

### Day 7: Snapshot Policy & Testing
- [ ] Implement snapshot policy
  - [ ] Size-based triggers
  - [ ] Time-based triggers
  - [ ] Manual triggers (API)
- [ ] Comprehensive testing
  - [ ] Large log compaction
  - [ ] Concurrent snapshot creation
  - [ ] Snapshot during active writes
- [ ] Performance benchmarks

**Files to create:**
```
pkg/raft/snapshot_policy.go
test/bench/snapshot_bench_test.go
```

**Week 7 Deliverables:**
- Snapshot creation and restoration
- Log compaction working
- InstallSnapshot RPC implemented
- Slow follower catch-up via snapshots
- Bounded log size
- Snapshot performance benchmarks

---

## Week 8: Dynamic Cluster Membership (Part 1)

### Day 1: Configuration Change Design
- [ ] Study Raft paper section 6
- [ ] Choose approach:
  - [ ] Joint consensus (C-old,new) - More complex, safer for multi-node changes
  - [ ] Single-server changes - Simpler, safer, recommended
- [ ] Design configuration log entries
- [ ] Document safety properties

**Files to create:**
```
docs/membership-changes.md
pkg/cluster/types.go
```

### Day 2: Configuration Log Entries
- [ ] Define configuration entry format
  - [ ] Node ID, address
  - [ ] Voting vs non-voting
  - [ ] Add vs remove operation
- [ ] Add configuration to log
- [ ] Implement configuration tracking
- [ ] Apply configuration changes

**Files to create:**
```
pkg/cluster/config.go
pkg/cluster/config_test.go
```

### Day 3: Add Node - Learner Phase
- [ ] Implement non-voting members (learners)
  - [ ] Add node as learner
  - [ ] Replicate log to learner
  - [ ] Don't count in quorum
  - [ ] Track catch-up progress
- [ ] Add learner tests

**Files to create:**
```
pkg/cluster/membership.go
pkg/cluster/learner.go
pkg/cluster/learner_test.go
```

### Day 4: Add Node - Promotion
- [ ] Implement learner promotion
  - [ ] Detect when learner caught up
  - [ ] Add configuration change entry
  - [ ] Promote to voting member
  - [ ] Update quorum calculation
- [ ] Test node addition flow

**Files to update:**
```
pkg/cluster/membership.go
pkg/raft/config_change.go (new)
test/integration/add_node_test.go (new)
```

### Day 5: Remove Node
- [ ] Implement node removal
  - [ ] Add configuration change entry
  - [ ] Remove from cluster
  - [ ] Update quorum calculation
  - [ ] Handle self-removal
  - [ ] Stop replication to removed node
- [ ] Test node removal

**Files to update:**
```
pkg/cluster/membership.go
pkg/raft/config_change.go
test/integration/remove_node_test.go (new)
```

### Day 6: Leader Transfer
- [ ] Implement leader transfer
  - [ ] Ensure target is caught up
  - [ ] Stop accepting client requests
  - [ ] Send TimeoutNow RPC
  - [ ] Step down
- [ ] Test leader transfer
- [ ] Use for graceful shutdown

**Files to create:**
```
pkg/raft/transfer.go
pkg/raft/transfer_test.go
```

### Day 7: Safety & Edge Cases
- [ ] Add safety checks
  - [ ] Prevent removing last node
  - [ ] One change at a time
  - [ ] Leader must be in new configuration
- [ ] Test edge cases
  - [ ] Concurrent change attempts
  - [ ] Leader failure during change
  - [ ] Partition during change
- [ ] Document guarantees

**Files to create:**
```
test/integration/membership_safety_test.go
docs/membership-safety.md
```

**Week 8 Deliverables:**
- Configuration change log entries
- Add node (learner → voting member)
- Remove node
- Leader transfer
- Safety guarantees verified

---

## Week 9: Dynamic Membership Complete & CLI

### Day 1: Replace Node Operation
- [ ] Implement replace node
  - [ ] Add new node as learner
  - [ ] Wait for catch-up
  - [ ] Promote new node
  - [ ] Remove old node
- [ ] Make atomic if possible
- [ ] Test replace scenarios

**Files to update:**
```
pkg/cluster/membership.go
test/integration/replace_node_test.go (new)
```

### Day 2: Cluster Configuration Queries
- [ ] Implement configuration API
  - [ ] List members
  - [ ] Get member status
  - [ ] Get leader
  - [ ] Get cluster health
- [ ] Add to client API
- [ ] Test queries

**Files to create:**
```
pkg/api/cluster.go
pkg/api/cluster_test.go
```

### Day 3: kvctl - Cluster Management
- [ ] Add cluster commands to kvctl
  ```bash
  kvctl members list
  kvctl members add <id> <address>
  kvctl members remove <id>
  kvctl leader
  kvctl status
  ```
- [ ] Add formatting (table output)
- [ ] Test CLI commands

**Files to update:**
```
cmd/kvctl/cluster.go (new)
cmd/kvctl/commands.go
```

### Day 4: kvctl - Data Operations
- [ ] Polish data commands
  ```bash
  kvctl put <key> <value>
  kvctl get <key>
  kvctl delete <key>
  kvctl scan [prefix]
  ```
- [ ] Add JSON output option
- [ ] Add batch import/export

**Files to update:**
```
cmd/kvctl/data.go (new)
cmd/kvctl/main.go
```

### Day 5: kvctl - Maintenance Operations
- [ ] Add maintenance commands
  ```bash
  kvctl snapshot create
  kvctl snapshot list
  kvctl compact
  kvctl backup
  kvctl restore
  ```
- [ ] Implement backup/restore
- [ ] Test maintenance ops

**Files to create:**
```
cmd/kvctl/maintenance.go (new)
pkg/api/maintenance.go (new)
```

### Day 6: Integration Testing - Full Membership
- [ ] Test complete membership scenarios
  - [ ] Start with 3 nodes
  - [ ] Add 2 nodes (scale to 5)
  - [ ] Remove 1 node (scale to 4)
  - [ ] Replace 1 node
  - [ ] Transfer leadership
- [ ] Verify data consistency throughout
- [ ] Test under load

**Files to create:**
```
test/integration/membership_full_test.go
```

### Day 7: Documentation & Examples
- [ ] Document membership operations
- [ ] Create operational runbook
  - [ ] How to add nodes
  - [ ] How to remove nodes
  - [ ] How to replace failed nodes
  - [ ] How to do rolling upgrades
- [ ] Add examples
- [ ] Update README

**Files to create:**
```
docs/operations.md
docs/examples/
examples/add-node.sh
examples/rolling-upgrade.sh
```

**Week 9 Deliverables:**
- Complete membership operations
- Full-featured kvctl CLI
- Cluster management API
- Operational documentation
- End-to-end membership tests

---

## Week 10: Production Features (Part 1)

### Day 1: Structured Logging
- [ ] Integrate Zap logger
  ```bash
  go get go.uber.org/zap
  ```
- [ ] Create logger package
  - [ ] Configure log levels
  - [ ] Add contextual fields
  - [ ] Support JSON and console output
- [ ] Replace all log statements
- [ ] Add request ID tracking

**Files to create:**
```
pkg/logging/logger.go
pkg/logging/context.go
```

### Day 2: Prometheus Metrics
- [ ] Integrate Prometheus client
  ```bash
  go get github.com/prometheus/client_golang
  ```
- [ ] Create metrics package
- [ ] Define key metrics:
  - [ ] Leader election count
  - [ ] Term number
  - [ ] Log size
  - [ ] Commit index
  - [ ] Applied index
  - [ ] Request latency (histogram)
  - [ ] Request throughput (counter)
  - [ ] Active connections (gauge)

**Files to create:**
```
pkg/metrics/prometheus.go
pkg/metrics/metrics.go
```

### Day 3: More Metrics & HTTP Endpoints
- [ ] Add more metrics:
  - [ ] Replication lag per follower
  - [ ] Snapshot frequency
  - [ ] WAL size
  - [ ] State machine size
  - [ ] RPC errors
  - [ ] Leadership changes
- [ ] Expose metrics HTTP endpoint
  - [ ] `/metrics` for Prometheus
  - [ ] `/health` for health checks
  - [ ] `/ready` for readiness

**Files to update:**
```
pkg/metrics/prometheus.go
cmd/keyval/main.go
```

### Day 4: Distributed Tracing Setup
- [ ] Integrate OpenTelemetry
  ```bash
  go get go.opentelemetry.io/otel
  go get go.opentelemetry.io/otel/exporters/jaeger
  ```
- [ ] Configure tracing
  - [ ] Set up Jaeger exporter
  - [ ] Create tracer provider
  - [ ] Add context propagation
- [ ] Create tracing helpers

**Files to create:**
```
pkg/tracing/tracer.go
pkg/tracing/context.go
```

### Day 5: Add Tracing Spans
- [ ] Instrument critical paths:
  - [ ] Client requests (start to finish)
  - [ ] Raft log append
  - [ ] State machine apply
  - [ ] Snapshot creation
  - [ ] RPC calls
- [ ] Add span attributes
- [ ] Link related spans

**Files to update:**
```
pkg/api/server.go
pkg/raft/node.go
pkg/raft/replication.go
pkg/statemachine/kv.go
```

### Day 6: Grafana Dashboard
- [ ] Create Grafana dashboard JSON
  - [ ] Cluster overview panel
  - [ ] Request rate/latency graphs
  - [ ] Log size trends
  - [ ] Replication lag
  - [ ] Leadership timeline
  - [ ] Error rate
- [ ] Document dashboard setup
- [ ] Add screenshots to docs

**Files to create:**
```
deployments/grafana/dashboard.json
docs/monitoring.md
```

### Day 7: Monitoring Integration Testing
- [ ] Test metrics collection
  - [ ] Verify all metrics exposed
  - [ ] Check metric accuracy
  - [ ] Test scraping
- [ ] Test tracing
  - [ ] Verify spans created
  - [ ] Check trace completeness
  - [ ] Test sampling
- [ ] Create docker-compose for monitoring stack
  - [ ] KeyVal cluster
  - [ ] Prometheus
  - [ ] Grafana
  - [ ] Jaeger

**Files to create:**
```
deployments/docker/docker-compose.yml
deployments/prometheus/prometheus.yml
test/integration/metrics_test.go
```

**Week 10 Deliverables:**
- Structured logging (Zap)
- Prometheus metrics
- Distributed tracing (OpenTelemetry)
- Grafana dashboard
- Full observability stack
- Monitoring documentation

---

## Week 11: Production Features (Part 2)

### Day 1: Configuration Management
- [ ] Create comprehensive config
  - [ ] YAML configuration file
  - [ ] Environment variable support
  - [ ] Command-line flag override
  - [ ] Validation
- [ ] Document all options
- [ ] Add examples

**Files to create:**
```
pkg/config/config.go
pkg/config/loader.go
configs/keyval.yaml.example
docs/configuration.md
```

### Day 2: Graceful Shutdown
- [ ] Implement shutdown logic
  - [ ] Stop accepting requests
  - [ ] Finish in-flight requests
  - [ ] Transfer leadership (if leader)
  - [ ] Flush WAL
  - [ ] Close connections
  - [ ] Cleanup resources
- [ ] Test shutdown scenarios
- [ ] Add timeout protection

**Files to create:**
```
pkg/server/shutdown.go
test/integration/shutdown_test.go
```

### Day 3: Rate Limiting & Backpressure
- [ ] Implement rate limiting
  - [ ] Per-client limits
  - [ ] Global cluster limits
  - [ ] Adaptive limiting
- [ ] Add backpressure handling
  - [ ] Slow client detection
  - [ ] Flow control
- [ ] Test under load

**Files to create:**
```
pkg/api/ratelimit.go
pkg/api/backpressure.go
test/integration/ratelimit_test.go
```

### Day 4: Docker Images
- [ ] Create Dockerfile
  - [ ] Multi-stage build
  - [ ] Minimal base image (alpine/distroless)
  - [ ] Non-root user
  - [ ] Health check
- [ ] Build scripts
- [ ] Test Docker image
- [ ] Push to registry

**Files to create:**
```
deployments/docker/Dockerfile
deployments/docker/build.sh
deployments/docker/.dockerignore
```

### Day 5: Kubernetes Manifests
- [ ] Create K8s manifests
  - [ ] StatefulSet for nodes
  - [ ] Service (headless)
  - [ ] ConfigMap
  - [ ] PersistentVolumeClaim
  - [ ] ServiceMonitor (Prometheus)
- [ ] Add Helm chart (optional)
- [ ] Test on local K8s (kind/minikube)

**Files to create:**
```
deployments/kubernetes/statefulset.yaml
deployments/kubernetes/service.yaml
deployments/kubernetes/configmap.yaml
deployments/kubernetes/pvc.yaml
docs/kubernetes-deployment.md
```

### Day 6: Security Hardening
- [ ] Add TLS support
  - [ ] Client-to-server TLS
  - [ ] Server-to-server TLS
  - [ ] Certificate validation
  - [ ] Mutual TLS (optional)
- [ ] Add authentication (basic)
  - [ ] Token-based auth
  - [ ] ACL system (basic)
- [ ] Test security features

**Files to create:**
```
pkg/security/tls.go
pkg/security/auth.go
docs/security.md
```

### Day 7: Performance Optimization
- [ ] Profile application
  - [ ] CPU profiling
  - [ ] Memory profiling
  - [ ] Goroutine analysis
  - [ ] Lock contention
- [ ] Optimize hot paths
  - [ ] Reduce allocations
  - [ ] Optimize serialization
  - [ ] Buffer pooling
  - [ ] Connection pooling
- [ ] Benchmark improvements

**Files to create:**
```
test/bench/profile_test.go
docs/performance-tuning.md
```

**Week 11 Deliverables:**
- Configuration management system
- Graceful shutdown
- Rate limiting
- Docker images
- Kubernetes deployment
- TLS and authentication
- Performance optimizations

---

## Week 12: Testing & Validation (Part 1)

### Day 1: Unit Test Coverage
- [ ] Audit test coverage
  ```bash
  make coverage
  ```
- [ ] Write missing unit tests
- [ ] Aim for >80% coverage
- [ ] Focus on edge cases
- [ ] Test error paths

**Goal:** >80% code coverage

### Day 2: Integration Test Expansion
- [ ] Add more integration tests
  - [ ] 5-node cluster tests
  - [ ] 7-node cluster tests
  - [ ] Asymmetric network delays
  - [ ] Slow disk simulation
  - [ ] High load scenarios
- [ ] Test failure combinations
- [ ] Verify recovery

**Files to create:**
```
test/integration/large_cluster_test.go
test/integration/network_test.go
test/integration/disk_test.go
```

### Day 3: Chaos Testing Framework
- [ ] Set up chaos testing
  - [ ] Random node crashes
  - [ ] Random network partitions
  - [ ] Random packet loss
  - [ ] Clock skew injection
- [ ] Create test scenarios
- [ ] Record history for verification

**Files to create:**
```
test/chaos/framework.go
test/chaos/scenarios.go
test/chaos/history.go
```

### Day 4: Jepsen-Style Tests
- [ ] Implement linearizability checker
  ```bash
  go get github.com/anishathalye/porcupine
  ```
- [ ] Run chaos tests with verification
  - [ ] Random operations
  - [ ] Random failures
  - [ ] Verify linearizability
- [ ] Test scenarios:
  - [ ] Partition during writes
  - [ ] Leader isolation
  - [ ] Majority partition
  - [ ] Bridge partition

**Files to create:**
```
test/chaos/linearizability.go
test/chaos/partition_test.go
test/chaos/isolation_test.go
```

### Day 5: Long-Running Stability Tests
- [ ] Create soak tests
  - [ ] Run for hours/days
  - [ ] Continuous operations
  - [ ] Random failures
  - [ ] Monitor for issues
- [ ] Memory leak detection
- [ ] Resource exhaustion tests
- [ ] Log analysis

**Files to create:**
```
test/chaos/soak_test.go
scripts/run-soak-test.sh
```

### Day 6: Performance Benchmarks
- [ ] Comprehensive benchmarks
  - [ ] Write throughput vs cluster size
  - [ ] Read throughput (ReadIndex vs lease)
  - [ ] Latency percentiles (p50, p95, p99, p999)
  - [ ] Scalability tests
- [ ] Compare with etcd
- [ ] Document results

**Files to create:**
```
test/bench/throughput_bench_test.go
test/bench/latency_bench_test.go
test/bench/scalability_bench_test.go
docs/benchmarks.md
```

### Day 7: Bug Fixing & Stabilization
- [ ] Fix bugs found during testing
- [ ] Address race conditions
- [ ] Fix edge cases
- [ ] Improve error handling
- [ ] Re-run all tests

**Week 12 Deliverables:**
- >80% unit test coverage
- Comprehensive integration tests
- Chaos testing framework
- Linearizability verification passing
- Performance benchmarks documented
- All critical bugs fixed

---

## Week 13: Final Polish & Release

### Day 1: Documentation Review
- [ ] Review all documentation
- [ ] Update README with final features
- [ ] Complete API documentation
- [ ] Write architecture docs
- [ ] Add diagrams and examples
- [ ] Proofread everything

**Files to review/update:**
```
README.md
docs/architecture.md (new)
docs/api-reference.md (new)
docs/operations.md
docs/troubleshooting.md (new)
```

### Day 2: Example Applications
- [ ] Create example apps
  - [ ] Simple counter service
  - [ ] Distributed lock
  - [ ] Session store
  - [ ] Configuration service
- [ ] Add to examples directory
- [ ] Document use cases

**Files to create:**
```
examples/counter/
examples/lock/
examples/session/
examples/config/
```

### Day 3: Operational Documentation
- [ ] Write operations guide
  - [ ] Installation
  - [ ] Configuration
  - [ ] Cluster setup
  - [ ] Monitoring
  - [ ] Backup/restore
  - [ ] Troubleshooting
  - [ ] Common issues
  - [ ] Performance tuning
- [ ] Create runbooks

**Files to create:**
```
docs/installation.md
docs/cluster-setup.md
docs/backup-restore.md
docs/troubleshooting.md
docs/runbooks/
```

### Day 4: Security Audit
- [ ] Review security
  - [ ] Code audit
  - [ ] Dependency audit
  - [ ] TLS configuration
  - [ ] Authentication/authorization
  - [ ] Input validation
- [ ] Document security best practices
- [ ] Add security.md

**Files to create:**
```
SECURITY.md
docs/security-best-practices.md
```

### Day 5: Performance Validation
- [ ] Final performance testing
  - [ ] Meet throughput targets (10,000 writes/sec)
  - [ ] Meet latency targets (<10ms p99)
  - [ ] Verify failover time (<2s)
  - [ ] Test scalability
- [ ] Document final numbers
- [ ] Create comparison charts

**Update:**
```
docs/benchmarks.md
README.md (performance section)
```

### Day 6: Release Preparation
- [ ] Version bump to v1.0.0
- [ ] Update CHANGELOG
- [ ] Create release notes
- [ ] Tag release
  ```bash
  git tag -a v1.0.0 -m "Production release"
  ```
- [ ] Build release binaries
  - [ ] Linux (amd64, arm64)
  - [ ] macOS (amd64, arm64)
  - [ ] Windows (amd64)
- [ ] Create GitHub release
- [ ] Push Docker images

**Files to create:**
```
RELEASE_NOTES.md
scripts/build-release.sh
```

### Day 7: Launch & Promotion
- [ ] Publish release
- [ ] Update repository
  - [ ] Final README polish
  - [ ] Add badges (build, coverage, etc.)
  - [ ] Add demo GIF/video
- [ ] Write blog post
  - [ ] Technical deep dive
  - [ ] Lessons learned
  - [ ] Performance results
- [ ] Share on social media
  - [ ] LinkedIn
  - [ ] Twitter/X
  - [ ] Reddit (r/golang, r/distributed)
  - [ ] Hacker News
- [ ] Add to portfolio

**Files to create:**
```
blog/building-raft-database.md
blog/lessons-learned.md
```

**Week 13 Deliverables:**
- Complete documentation
- Example applications
- Operational runbooks
- Security audit complete
- v1.0.0 release published
- Blog posts and promotion

---

## Post-Launch: Continuous Improvement

### Optional Enhancements

**Advanced Features:**
- [ ] Multi-Raft (sharding)
- [ ] Batched log replication
- [ ] Pre-vote optimization
- [ ] Leadership transfer on network partition
- [ ] Learner optimizations
- [ ] Compressed snapshots
- [ ] Incremental snapshots

**Additional Testing:**
- [ ] Formal TLA+ specification
- [ ] Model checking with TLC
- [ ] Fuzz testing
- [ ] Property-based testing

**Performance:**
- [ ] Zero-copy optimizations
- [ ] DPDK for networking (extreme)
- [ ] io_uring for disk I/O
- [ ] Faster serialization (cap'n proto)

**Ecosystem:**
- [ ] Client libraries (Python, Java, JavaScript)
- [ ] Prometheus exporter
- [ ] Terraform provider
- [ ] Kubernetes operator

---

## Success Metrics

### Functional Requirements
- [x] Leader election works reliably
- [ ] Log replication maintains consistency
- [ ] Linearizable reads and writes
- [ ] Survives network partitions
- [ ] Crash recovery works
- [ ] Snapshots prevent unbounded growth
- [ ] Dynamic membership works

### Performance Requirements
- [ ] 10,000+ writes/second (3-node cluster) ⚡
- [ ] <5ms p50 write latency
- [ ] <10ms p99 write latency
- [ ] <1ms read latency (with leases)
- [ ] <2 second failover time

### Quality Requirements
- [ ] >80% code coverage
- [ ] Zero data loss in chaos tests
- [ ] Linearizability verification passes
- [ ] No race conditions (go test -race)
- [ ] Memory leak free (multi-hour soak tests)

### Documentation Requirements
- [x] Comprehensive README
- [ ] API documentation
- [ ] Operations guide
- [ ] Architecture documentation
- [ ] Troubleshooting guide

---

## Interview Preparation Checklist

As you build, prepare to discuss:

### Technical Understanding
- [ ] Explain Raft algorithm in detail
- [ ] Describe leader election process
- [ ] Explain log replication guarantees
- [ ] Discuss CAP theorem tradeoffs
- [ ] Compare Raft vs Paxos vs other consensus algorithms

### Design Decisions
- [ ] Why Go for this project?
- [ ] Why gRPC vs custom protocol?
- [ ] BoltDB vs other storage engines
- [ ] Single-server vs joint consensus for membership
- [ ] ReadIndex vs lease-based reads

### Failure Scenarios
- [ ] What happens during network partition?
- [ ] How to handle split-brain?
- [ ] Recovery from data corruption
- [ ] Handling slow/failed nodes
- [ ] Byzantine fault tolerance (why Raft doesn't handle it)

### Performance
- [ ] Latency vs throughput tradeoffs
- [ ] How to optimize write path?
- [ ] How to optimize read path?
- [ ] Batch vs individual operations
- [ ] Scalability limits

### Production Operations
- [ ] How to monitor the cluster?
- [ ] How to add/remove nodes safely?
- [ ] Backup and disaster recovery
- [ ] Upgrade strategy (rolling upgrades)
- [ ] Capacity planning

---

## Resources

### Essential Reading
- **Raft Paper**: https://raft.github.io/raft.pdf
- **Raft PhD Thesis**: https://github.com/ongardie/dissertation
- **etcd Documentation**: https://etcd.io/docs/
- **DDIA Book**: Chapter 9 (Consistency and Consensus)

### Reference Implementations
- **etcd/raft**: https://github.com/etcd-io/raft
- **hashicorp/raft**: https://github.com/hashicorp/raft
- **CockroachDB**: https://github.com/cockroachdb/cockroach

### Testing & Verification
- **Jepsen**: https://jepsen.io/
- **Porcupine**: https://github.com/anishathalye/porcupine
- **TLA+**: https://lamport.azurewebsites.net/tla/tla.html

### Tools
- **Protocol Buffers**: https://protobuf.dev/
- **gRPC**: https://grpc.io/
- **BoltDB**: https://github.com/etcd-io/bbolt
- **Prometheus**: https://prometheus.io/
- **Grafana**: https://grafana.com/

---

## Daily Development Routine

**Recommended workflow:**

1. **Morning** (2-3 hours)
   - Review previous day's work
   - Read relevant Raft paper sections
   - Write code for current task
   - Write tests first (TDD)

2. **Afternoon** (2-3 hours)
   - Complete implementation
   - Run tests and fix issues
   - Code review (self-review)
   - Commit with good messages

3. **Evening** (optional, 1 hour)
   - Read reference implementations
   - Study related topics
   - Plan next day's work
   - Update progress tracking

**Weekly:**
- Review week's progress
- Run all tests including integration
- Update documentation
- Plan next week

---

## Completion Certificate

Once you complete this project, you will have:

✅ Built a production-grade distributed database
✅ Implemented full Raft consensus algorithm
✅ Mastered distributed systems concepts
✅ Created a portfolio-worthy project
✅ Prepared for senior engineer interviews
✅ Learned technologies used in production systems

**Congratulations on taking on this ambitious journey!** 🚀

This project demonstrates the kind of deep technical expertise that distinguishes senior engineers from junior developers. Stick with it, and you'll have one of the most impressive projects in your portfolio.

---

**Current Status:** Week 1 Complete ✅
**Next Up:** Week 2 - Core Raft Structures & Leader Election

Let's build something amazing! 💪
