# Contributing to KeyVal

Thank you for your interest in contributing to KeyVal! This document provides guidelines for contributing to this distributed database project.

## Getting Started

1. **Read the Documentation**
   - [README.md](README.md) - Project overview
   - [ROADMAP.md](ROADMAP.md) - Development plan
   - [QUICK_START.md](QUICK_START.md) - Setup guide

2. **Set Up Development Environment**
   ```bash
   git clone https://github.com/therealutkarshpriyadarshi/keyval.git
   cd keyval
   make init
   make deps
   make test
   ```

3. **Understand the Architecture**
   - Read the Raft paper: https://raft.github.io/raft.pdf
   - Explore etcd/raft implementation
   - Review the codebase structure

## Development Workflow

### 1. Pick a Task

Choose from:
- Issues labeled `good-first-issue`
- Features in the roadmap
- Bug fixes
- Documentation improvements
- Performance optimizations

### 2. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

### 3. Write Code

**Best Practices:**
- Follow Go conventions and idioms
- Write clear, self-documenting code
- Add comments for complex logic
- Keep functions small and focused
- Use meaningful variable names
- Handle errors explicitly

**Code Style:**
```bash
# Format code
make fmt

# Run linter
make lint

# Fix common issues
golangci-lint run --fix
```

### 4. Write Tests

**Test Requirements:**
- Unit tests for all new functions
- Integration tests for components
- Benchmark tests for performance-critical code
- Aim for >80% code coverage

**Running Tests:**
```bash
# Unit tests
make test

# Integration tests
make integration-test

# Coverage
make coverage

# Specific package
go test -v ./pkg/raft/...
```

**Test Structure:**
```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    interface{}
        wantErr bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 5. Commit Changes

**Commit Message Format:**
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Adding tests
- `perf`: Performance improvements
- `chore`: Build process or auxiliary tool changes

**Examples:**
```
feat(raft): implement leader election
fix(storage): handle corruption in WAL
docs(readme): update installation instructions
test(replication): add log conflict test
```

### 6. Submit Pull Request

1. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a PR with:
   - Clear title and description
   - Reference to related issues
   - Summary of changes
   - Test results
   - Performance impact (if applicable)

3. Ensure CI passes:
   - All tests pass
   - Code coverage maintained
   - Linter checks pass
   - Build succeeds

## Code Review Process

### For Contributors

- Be responsive to feedback
- Address all comments
- Keep PR scope focused
- Update PR description as needed
- Be patient and respectful

### For Reviewers

- Be constructive and respectful
- Focus on code quality and correctness
- Check for:
  - Correctness
  - Performance
  - Security
  - Test coverage
  - Documentation
  - Code style

## Specific Guidelines

### Raft Implementation

**Critical Rules:**
- Never violate Raft safety properties
- Follow the paper's algorithm precisely
- Document any deviations
- Add comprehensive tests

**Safety Properties:**
- Election Safety: At most one leader per term
- Leader Append-Only: Leader never overwrites log
- Log Matching: Identical entries at same index
- Leader Completeness: Committed entries in future leaders
- State Machine Safety: Same command at same index

### Concurrency

- Use proper synchronization (mutexes, channels)
- Avoid race conditions
- Test with `-race` flag
- Document locking strategy
- Minimize lock contention

### Performance

- Profile before optimizing
- Benchmark critical paths
- Document performance characteristics
- Avoid premature optimization
- Consider memory allocations

### Error Handling

```go
// Good
if err != nil {
    return fmt.Errorf("failed to replicate log: %w", err)
}

// Bad
if err != nil {
    panic(err)
}
```

### Logging

```go
// Use structured logging
logger.Info("leader elected",
    zap.String("nodeID", nodeID),
    zap.Int64("term", term))

// Avoid
log.Println("leader elected") // No structure
```

## Testing Guidelines

### Unit Tests

- Test each function in isolation
- Use table-driven tests
- Mock external dependencies
- Test edge cases
- Test error conditions

### Integration Tests

- Test component interactions
- Use real dependencies where possible
- Test failure scenarios
- Verify correctness properties

### Chaos Tests

- Test with random failures
- Verify linearizability
- Test network partitions
- Test concurrent operations
- Use Jepsen-style testing

## Documentation

### Code Documentation

```go
// Package raft implements the Raft consensus algorithm.
package raft

// Node represents a single Raft node in the cluster.
// It maintains the node's state and handles RPCs from
// other nodes and clients.
type Node struct {
    // Persistent state
    currentTerm int64
    votedFor    string
    log         []LogEntry

    // Volatile state
    commitIndex int64
    lastApplied int64
}

// RequestVote handles RequestVote RPC from candidates.
// It grants votes according to Raft's voting rules.
func (n *Node) RequestVote(req *RequestVoteRequest) (*RequestVoteResponse, error) {
    // Implementation
}
```

### User Documentation

- Clear and concise
- Include examples
- Cover common use cases
- Explain concepts
- Provide troubleshooting tips

## Performance Benchmarks

```go
func BenchmarkLogAppend(b *testing.B) {
    log := NewLog()
    entry := LogEntry{Term: 1, Data: []byte("test")}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        log.Append(entry)
    }
}
```

## Release Process

1. Update version numbers
2. Update CHANGELOG.md
3. Run full test suite
4. Build release binaries
5. Tag release
6. Update documentation
7. Announce release

## Getting Help

- Open an issue for questions
- Join discussions in issues/PRs
- Read the Raft paper and etcd docs
- Check reference implementations

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Welcome newcomers
- Share knowledge
- Collaborate openly

## Recognition

Contributors will be:
- Listed in README.md
- Credited in release notes
- Acknowledged in commit history

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for contributing to KeyVal! Your efforts help make distributed systems more accessible and understandable.
