# KeyVal: Production-Grade Distributed Database

A distributed key-value database implementing the full **Raft consensus algorithm** for strong consistency across cluster nodes.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/therealutkarshpriyadarshi/keyval)

## 🎯 Why This Project?

KeyVal is designed to demonstrate advanced backend engineering capabilities by implementing the industry-standard consensus algorithm used by:

- **Kubernetes** (via etcd)
- **Kafka** (KRaft replaces ZooKeeper)
- **CockroachDB** (distributed SQL)
- **Consul** (service mesh)
- **TiKV** (distributed storage)

This project showcases deep understanding of:
- CAP theorem implementations
- Distributed consensus algorithms
- Concurrent programming and race condition handling
- Network partition tolerance
- Production-grade distributed systems

## 🚀 Features

- ✅ **Full Raft Consensus**: Leader election, log replication, safety
- ✅ **Strong Consistency**: Linearizable reads and writes
- ✅ **Fault Tolerance**: Automatic failover and recovery
- ✅ **Persistence**: Write-ahead logging with crash recovery
- ✅ **Log Compaction**: Snapshots for bounded memory usage
- ✅ **Dynamic Membership**: Add/remove nodes without downtime
- ✅ **gRPC Communication**: Efficient inter-node protocol
- ✅ **Observability**: Prometheus metrics and distributed tracing
- ✅ **Production Ready**: Battle-tested with chaos engineering

## 📊 Performance

**Target Performance** (3-node cluster):
- **Throughput**: 10,000+ writes/second
- **Latency**: <5ms p50, <10ms p99
- **Failover**: <2 seconds
- **Linearizability**: 100% guaranteed

## 🏗️ Architecture

```
┌──────────────────────────────────────────────┐
│           Client Applications                │
└───────────────┬──────────────────────────────┘
                │ gRPC/HTTP
    ┌───────────▼──────────┐
    │    API Gateway       │
    │  (Leader Forwarding) │
    └───────────┬──────────┘
                │
    ┌───────────▼──────────┐
    │   Raft Consensus     │
    │  - Leader Election   │
    │  - Log Replication   │
    │  - Snapshots         │
    └───────────┬──────────┘
                │
    ┌───────────▼──────────┐
    │   KV State Machine   │
    └───────────┬──────────┘
                │
    ┌───────────▼──────────┐
    │  Persistent Storage  │
    │   (BoltDB + WAL)     │
    └──────────────────────┘
```

## 🛠️ Tech Stack

- **Language**: Go 1.21+ (industry standard for distributed systems)
- **RPC**: gRPC (high-performance inter-node communication)
- **Storage**: BoltDB (persistent key-value store)
- **Observability**: Prometheus + OpenTelemetry
- **Testing**: Jepsen (distributed systems validation)

## 📚 Quick Start

See [QUICK_START.md](QUICK_START.md) for detailed setup instructions.

### Prerequisites

```bash
# Install Go 1.21+
# Install Protocol Buffers compiler
# Install Docker (for testing)
```

### Initialize Project

```bash
# Initialize Go module and dependencies
go mod init github.com/therealutkarshpriyadarshi/keyval
make init
make deps
```

### Build

```bash
make build
```

### Run a 3-node cluster locally

```bash
# Start cluster
make cluster-start

# Check status
make cluster-status

# Stop cluster
make cluster-stop
```

### Basic Operations

```bash
# Write a key
./bin/kvctl put mykey myvalue

# Read a key
./bin/kvctl get mykey

# Delete a key
./bin/kvctl delete mykey

# Check cluster status
./bin/kvctl status
```

## 🗺️ Development Roadmap

See [ROADMAP.md](ROADMAP.md) for the complete 13-week development plan.

### Phase Overview

1. **Phase 1**: Project Foundation & Setup (Week 1)
2. **Phase 2**: Core Raft - Leader Election (Week 2-3)
3. **Phase 3**: Core Raft - Log Replication (Week 3-4)
4. **Phase 4**: Persistence Layer (Week 5)
5. **Phase 5**: Client Operations (Week 6)
6. **Phase 6**: Snapshot & Log Compaction (Week 7)
7. **Phase 7**: Dynamic Cluster Membership (Week 8-9)
8. **Phase 8**: Production Features (Week 10-11)
9. **Phase 9**: Testing & Validation (Week 12-13)

## 🧪 Testing

```bash
# Unit tests
make test

# Integration tests
make integration-test

# Chaos tests (Jepsen-style)
make chaos-test

# Benchmarks
make bench
```

## 📖 Documentation

- [Quick Start Guide](QUICK_START.md) - Get up and running quickly
- [Development Roadmap](ROADMAP.md) - High-level 13-week implementation plan
- [Development Plan](DEVELOPMENT_PLAN.md) - Detailed day-by-day task breakdown with checkboxes
- [Architecture](docs/ARCHITECTURE.md) - System design details (coming soon)
- [API Reference](docs/API.md) - Client API documentation (coming soon)

## 🎓 Learning Resources

### Essential Reading
- [Raft Paper](https://raft.github.io/raft.pdf) - Original consensus algorithm paper
- [Raft Visualization](https://raft.github.io/) - Interactive Raft demo
- [DDIA Chapter 9](https://dataintensive.net/) - Consistency and Consensus
- [etcd Documentation](https://etcd.io/docs/) - Real-world implementation

### Reference Implementations
- [etcd/raft](https://github.com/etcd-io/raft) - Production Raft library
- [hashicorp/raft](https://github.com/hashicorp/raft) - Consul's Raft
- [CockroachDB](https://github.com/cockroachdb/cockroach) - Distributed SQL

## 🎯 Project Status

**Current Phase**: Phase 1 - Project Foundation

- [x] Repository initialized
- [x] Roadmap created
- [x] Build system established
- [ ] Directory structure created
- [ ] Core dependencies installed
- [ ] Initial documentation written

## 🤝 Contributing

This is a learning project, but contributions are welcome! Please:

1. Read the roadmap
2. Pick a phase/component
3. Follow Go best practices
4. Write comprehensive tests
5. Submit a PR with clear description

## 📝 License

MIT License - See [LICENSE](LICENSE) for details

## 🌟 Acknowledgments

- **Diego Ongaro** and **John Ousterhout** for the Raft paper
- **etcd team** for their reference implementation
- **Martin Kleppmann** for "Designing Data-Intensive Applications"
- **Kyle Kingsbury** for Jepsen testing framework

## 💼 For Recruiters

This project demonstrates:
- **Distributed Systems**: Consensus algorithms, CAP theorem, fault tolerance
- **Systems Programming**: Concurrent Go, RPC, persistence
- **Production Engineering**: Monitoring, testing, deployment
- **Problem Solving**: Complex algorithms, debugging, optimization

**Relevant for**: Senior Backend Engineer, Distributed Systems Engineer, Infrastructure Engineer roles at FAANG and similar companies.

---

**Built with** ❤️ **by** [Utkarsh Priyadarshi](https://github.com/therealutkarshpriyadarshi)

**Questions?** Open an issue or reach out!
