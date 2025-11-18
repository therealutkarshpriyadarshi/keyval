# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure
- Comprehensive roadmap for 13-week development plan
- Quick start guide for developers
- Makefile with build, test, and development targets
- GitHub Actions CI/CD pipeline
- MIT License
- Contributing guidelines
- Project documentation (README, ROADMAP, QUICK_START)
- Go module initialization
- Directory structure for all phases

### Phase 1: Foundation (In Progress)
- [x] Project repository setup
- [x] Documentation structure
- [x] Build system (Makefile)
- [x] CI/CD pipeline
- [x] Go module initialization
- [ ] Core dependencies installation
- [ ] Protocol buffer definitions
- [ ] Basic configuration management

### Planned

#### Phase 2: Leader Election (Week 2-3)
- Raft node structure and state management
- RequestVote RPC implementation
- Election timer with randomization
- Term management
- Candidate voting logic

#### Phase 3: Log Replication (Week 3-4)
- Log entry structure
- AppendEntries RPC implementation
- Heartbeat mechanism
- Log consistency checking
- Commit index advancement

#### Phase 4: Persistence (Week 5)
- BoltDB integration
- Write-ahead log (WAL)
- Crash recovery
- Fsync durability guarantees

#### Phase 5: Client Operations (Week 6)
- Key-value state machine
- Client API (Get, Put, Delete)
- Leader-only writes
- Linearizable reads

#### Phase 6: Snapshots (Week 7)
- Snapshot creation and installation
- Log compaction
- InstallSnapshot RPC

#### Phase 7: Dynamic Membership (Week 8-9)
- Configuration change log entries
- Node addition/removal
- Non-voting members (learners)

#### Phase 8: Production Features (Week 10-11)
- Prometheus metrics
- OpenTelemetry tracing
- CLI tool (kvctl)
- Docker images
- Kubernetes manifests

#### Phase 9: Testing & Validation (Week 12-13)
- Integration tests
- Chaos/Jepsen tests
- Performance benchmarks
- Linearizability verification

## [0.1.0] - 2024-11-18

### Added
- Initial repository setup
- Project documentation
- Development infrastructure

---

## Version History

- **v0.1.0** (2024-11-18): Initial setup and documentation
- **v0.2.0** (Planned): Phase 1 complete - Foundation
- **v0.3.0** (Planned): Phase 2 complete - Leader Election
- **v0.4.0** (Planned): Phase 3 complete - Log Replication
- **v0.5.0** (Planned): Phase 4 complete - Persistence
- **v0.6.0** (Planned): Phase 5 complete - Client Operations
- **v0.7.0** (Planned): Phase 6 complete - Snapshots
- **v0.8.0** (Planned): Phase 7 complete - Dynamic Membership
- **v0.9.0** (Planned): Phase 8 complete - Production Features
- **v1.0.0** (Planned): Phase 9 complete - Production Ready

## Notes

This is a learning project to demonstrate advanced distributed systems knowledge.
The roadmap is ambitious and may take 3-4 months to complete all phases.

For detailed implementation plans, see [ROADMAP.md](ROADMAP.md).
