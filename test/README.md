# KeyVal Test Suite

This directory contains the comprehensive test suite for the KeyVal distributed database.

## Directory Structure

```
test/
├── README.md              # This file
├── testutil/              # Test utilities and helpers
│   ├── cluster.go         # Cluster management utilities
│   ├── assertions.go      # Test assertion helpers
│   └── chaos.go           # Chaos testing utilities
├── integration/           # Integration tests
│   ├── election_test.go   # Leader election tests
│   ├── crash_test.go      # Crash recovery tests
│   ├── client_test.go     # Client operations tests
│   ├── linearizability_test.go  # Linearizability tests
│   ├── log_conflict_test.go     # Log conflict resolution tests
│   └── raft_cluster_test.go     # Cluster behavior tests
├── e2e/                   # End-to-end tests
│   └── full_workflow_test.go    # Complete workflow tests
├── chaos/                 # Chaos/fault injection tests
│   └── chaos_test.go      # Chaos scenarios
└── bench/                 # Benchmarks
    └── persistence_bench_test.go  # Performance benchmarks
```

## Quick Start

### Run All Unit Tests

```bash
make test
```

### Run Integration Tests

```bash
make integration-test
```

### Run E2E Tests

```bash
go test -v -tags=e2e -timeout=15m ./test/e2e/...
```

### Run Chaos Tests

```bash
make chaos-test
```

### Run Full Test Suite

```bash
./scripts/test.sh
```

## Test Utilities

The `testutil` package provides helpers for writing tests:

### Cluster Management

```go
import "github.com/therealutkarshpriyadarshi/keyval/test/testutil"

// Create a test cluster
config := testutil.FastClusterConfig()
config.Size = 3
cluster := testutil.NewTestCluster(t, config)
defer cluster.Stop()

cluster.Start()
leader := cluster.WaitForLeader(5 * time.Second)
```

### Assertions

```go
testutil.AssertLeaderElected(t, cluster, timeout)
testutil.AssertSingleLeader(t, cluster)
testutil.AssertEqual(t, expected, actual, "message")
testutil.AssertEventually(t, condition, timeout, "message")
```

### Chaos Testing

```go
chaos := testutil.NewChaosScheduler(t, cluster)
chaos.Start(2 * time.Second)
defer chaos.Stop()
```

## Test Categories

### Unit Tests (`pkg/*_test.go`)

Test individual components in isolation.

**Build tags**: None
**Timeout**: Fast (< 30s)

### Integration Tests (`test/integration/*_test.go`)

Test interactions between components, especially Raft cluster behavior.

**Build tags**: `// +build integration`
**Timeout**: Medium (< 10m)

### E2E Tests (`test/e2e/*_test.go`)

Test complete user workflows from API to state machine.

**Build tags**: `// +build e2e`
**Timeout**: Long (< 15m)

### Chaos Tests (`test/chaos/*_test.go`)

Test system behavior under various failure conditions.

**Build tags**: `// +build chaos`
**Timeout**: Very Long (< 30m)

## Configuration

### Cluster Configs

**FastClusterConfig**: For quick tests
- Election timeout: 50-100ms
- Heartbeat: 20ms

**DefaultClusterConfig**: For realistic tests
- Election timeout: 150-300ms
- Heartbeat: 50ms

### Example

```go
// Fast config for unit/integration tests
config := testutil.FastClusterConfig()

// Default config for E2E tests
config := testutil.DefaultClusterConfig()

// Custom config
config := &testutil.ClusterConfig{
    Size:               5,
    ElectionTimeoutMin: 100 * time.Millisecond,
    ElectionTimeoutMax: 200 * time.Millisecond,
    HeartbeatInterval:  30 * time.Millisecond,
}
```

## Writing New Tests

### 1. Choose Test Type

- **Unit**: Testing single function/method
- **Integration**: Testing component interactions
- **E2E**: Testing full workflows
- **Chaos**: Testing failure scenarios

### 2. Use Appropriate Build Tags

```go
// For integration tests
// +build integration

package integration

// For E2E tests
// +build e2e

package e2e

// For chaos tests
// +build chaos

package chaos
```

### 3. Follow Naming Conventions

- `TestComponentName_Scenario` for unit tests
- `TestRaftCluster_Scenario` for integration tests
- `TestE2E_Scenario` for E2E tests
- `TestChaos_Scenario` for chaos tests

### 4. Use Test Utilities

Leverage `testutil` package for:
- Cluster setup/teardown
- Assertions
- Chaos injection
- Workload generation

### 5. Example Test

```go
// +build integration

package integration

import (
    "testing"
    "time"
    "github.com/therealutkarshpriyadarshi/keyval/test/testutil"
)

func TestRaftCluster_NewScenario(t *testing.T) {
    // Setup
    config := testutil.FastClusterConfig()
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()

    // Execute
    cluster.Start()
    leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)

    // Verify
    testutil.AssertSingleLeader(t, cluster)

    // Additional test logic...
}
```

## Coverage

Coverage reports are generated with:

```bash
make coverage
```

This creates:
- `coverage.out`: Coverage data
- `coverage.html`: Interactive HTML report

**Minimum coverage threshold**: 60%

## CI/CD

Tests run automatically on:
- Every push to main/develop branches
- Every pull request
- Nightly chaos tests (scheduled)

See `.github/workflows/` for CI configuration.

## Troubleshooting

### Tests Hanging

1. Check for missing cleanup (`defer cluster.Stop()`)
2. Verify timeouts are set
3. Look for goroutine leaks

### Flaky Tests

1. Run with race detector: `go test -race`
2. Check for timing assumptions
3. Use `AssertEventually` instead of fixed sleeps
4. Review synchronization

### Performance Issues

1. Use `FastClusterConfig` for faster tests
2. Run tests in parallel where safe: `t.Parallel()`
3. Skip long tests in short mode: `if testing.Short() { t.Skip() }`

## Best Practices

1. **Cleanup**: Always use `defer` for cleanup
2. **Timeouts**: Set reasonable timeouts for all operations
3. **Independence**: Tests should not depend on each other
4. **Determinism**: Avoid flaky tests with proper synchronization
5. **Documentation**: Comment complex test scenarios
6. **Assertions**: Use descriptive messages in assertions

## Resources

- [Testing Guide](../docs/TESTING.md) - Comprehensive testing documentation
- [README](../README.md) - Project overview
- [Development Plan](../DEVELOPMENT_PLAN.md) - Implementation roadmap

## Contributing

When adding new tests:

1. Follow existing patterns and conventions
2. Use test utilities for consistency
3. Add appropriate build tags
4. Document complex scenarios
5. Ensure tests are deterministic
6. Run full test suite before submitting PR

```bash
# Before submitting PR
./scripts/test.sh
```
