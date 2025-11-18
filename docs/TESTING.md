# Testing Guide for KeyVal

This document describes the comprehensive testing strategy for the KeyVal distributed database project.

## Table of Contents

1. [Overview](#overview)
2. [Test Types](#test-types)
3. [Running Tests](#running-tests)
4. [Test Utilities](#test-utilities)
5. [Writing Tests](#writing-tests)
6. [CI/CD Integration](#cicd-integration)
7. [Coverage Requirements](#coverage-requirements)
8. [Best Practices](#best-practices)

## Overview

KeyVal employs a multi-layered testing strategy to ensure reliability and correctness:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test interactions between components
- **End-to-End Tests**: Test complete workflows in a full cluster
- **Chaos Tests**: Test system behavior under failure conditions
- **Benchmarks**: Measure performance characteristics

## Test Types

### Unit Tests

Unit tests validate individual functions and methods in isolation.

**Location**: `pkg/*_test.go`

**Run with**:
```bash
make test
# or
go test -v -race -short ./...
```

**Example**:
```go
func TestKVStateMachine_Put(t *testing.T) {
    sm := statemachine.NewKVStateMachine()
    err := sm.Put("key1", []byte("value1"))
    testutil.AssertNoError(t, err, "PUT should succeed")
}
```

### Integration Tests

Integration tests validate interactions between multiple components, particularly Raft cluster behavior.

**Location**: `test/integration/*_test.go`

**Build tag**: `// +build integration`

**Run with**:
```bash
make integration-test
# or
go test -v -tags=integration -timeout=10m ./test/integration/...
```

**Example**:
```go
func TestRaftCluster_BasicElection(t *testing.T) {
    config := testutil.FastClusterConfig()
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()

    cluster.Start()
    leader := testutil.AssertLeaderElected(t, cluster, 3*time.Second)
    testutil.AssertSingleLeader(t, cluster)
}
```

### End-to-End Tests

E2E tests validate complete user workflows from API to state machine.

**Location**: `test/e2e/*_test.go`

**Build tag**: `// +build e2e`

**Run with**:
```bash
go test -v -tags=e2e -timeout=15m ./test/e2e/...
```

**Example**:
```go
func TestE2E_FullWorkflow(t *testing.T) {
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()

    cluster.Start()
    leader := testutil.AssertLeaderElected(t, cluster, 5*time.Second)

    server := api.NewServer(leader)
    defer server.Stop()

    // Test PUT, GET, DELETE operations
}
```

### Chaos Tests

Chaos tests validate system resilience under various failure scenarios.

**Location**: `test/chaos/*_test.go`

**Build tag**: `// +build chaos`

**Run with**:
```bash
make chaos-test
# or
go test -v -tags=chaos -timeout=30m ./test/chaos/...
```

**Scenarios tested**:
- Random node failures
- Network partitions
- Cascading failures
- High load with failures
- Real-world failure patterns

**Example**:
```go
func TestChaos_RandomFailures(t *testing.T) {
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()

    chaos := testutil.NewChaosScheduler(t, cluster)
    chaos.Start(2 * time.Second)
    defer chaos.Stop()

    // Run workload during chaos...
}
```

### Benchmarks

Benchmarks measure performance characteristics.

**Location**: `test/bench/*_test.go`

**Run with**:
```bash
make bench
# or
go test -bench=. -benchmem ./test/bench/...
```

## Running Tests

### Quick Test

Run unit tests only:
```bash
make test
```

### Full Test Suite

Run all tests including integration and E2E:
```bash
./scripts/test.sh
```

### Test with Coverage

Generate coverage report:
```bash
make coverage
# Opens coverage.html in browser
```

### Specific Test Package

```bash
go test -v ./pkg/raft/...
```

### Specific Test Function

```bash
go test -v -run TestRaftCluster_BasicElection ./test/integration/...
```

### Short Mode

Skip long-running tests:
```bash
go test -short ./...
```

### Race Detector

Run tests with race detector (slower but catches concurrency bugs):
```bash
go test -race ./...
```

## Test Utilities

The `test/testutil` package provides comprehensive helpers for writing tests.

### Cluster Management

```go
// Create a test cluster
config := testutil.FastClusterConfig()
cluster := testutil.NewTestCluster(t, config)
defer cluster.Stop()

cluster.Start()

// Wait for leader
leader := cluster.WaitForLeader(5 * time.Second)

// Get specific node
node := cluster.GetNode(0)

// Stop/start nodes
cluster.StopNode(0)
cluster.StartNode(0)

// Wait for convergence
cluster.WaitForConvergence(5 * time.Second)
```

### Assertions

```go
// Leader assertions
testutil.AssertLeaderElected(t, cluster, timeout)
testutil.AssertSingleLeader(t, cluster)
testutil.AssertNoLeader(t, cluster)

// Value assertions
testutil.AssertEqual(t, expected, actual, "message")
testutil.AssertNotEqual(t, unexpected, actual, "message")
testutil.AssertTrue(t, condition, "message")
testutil.AssertFalse(t, condition, "message")

// Error assertions
testutil.AssertNoError(t, err, "message")
testutil.AssertError(t, err, "message")

// Eventual consistency
testutil.AssertEventually(t, func() bool {
    return condition()
}, timeout, "message")
```

### Chaos Testing

```go
// Create chaos scheduler
chaos := testutil.NewChaosScheduler(t, cluster)
chaos.Start(2 * time.Second) // Chaos every 2 seconds
defer chaos.Stop()

// Network partitions
partition := testutil.NetworkPartition{
    Group1: []int{0, 1},
    Group2: []int{2, 3, 4},
}
testutil.CreateNetworkPartition(cluster, partition.Group1, partition.Group2)
testutil.HealNetworkPartition(cluster, partition)

// Simulate specific failures
testutil.SimulateCrash(t, cluster, nodeIndex)
```

### Workload Generation

```go
workload := testutil.NewRandomWorkload(t, cluster)
workload.Start(10) // 10 ops/second
defer workload.Stop()
```

## Writing Tests

### Test Structure

Follow this structure for consistency:

```go
func TestFeatureName_Scenario(t *testing.T) {
    // Setup
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

### Test Naming

- Use descriptive names: `TestComponent_Scenario`
- Integration tests: `TestRaftCluster_LeaderFailover`
- E2E tests: `TestE2E_FullWorkflow`
- Chaos tests: `TestChaos_RandomFailures`

### Table-Driven Tests

For testing multiple scenarios:

```go
func TestConfiguration_Validation(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {"valid config", validConfig, false},
        {"invalid timeout", invalidConfig, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Cleanup

Always use defer for cleanup:

```go
func TestExample(t *testing.T) {
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()  // Ensures cleanup even if test fails

    server := api.NewServer(leader)
    defer server.Stop()
}
```

### Test Helpers

Mark helper functions with `t.Helper()`:

```go
func setupTestCluster(t *testing.T) *testutil.TestCluster {
    t.Helper()
    config := testutil.FastClusterConfig()
    cluster := testutil.NewTestCluster(t, config)
    cluster.Start()
    return cluster
}
```

## CI/CD Integration

### GitHub Actions Workflows

**Main CI Pipeline** (`.github/workflows/ci.yml`):
- Runs on every push and PR
- Linting, unit tests, integration tests, E2E tests
- Coverage reporting with threshold enforcement (60%)
- Multi-OS builds (Linux, macOS) and architectures (amd64, arm64)

**Chaos Testing** (`.github/workflows/chaos.yml`):
- Runs nightly at 2 AM UTC
- Can be manually triggered
- Long-running chaos and stress tests

### Coverage Reporting

Coverage is automatically:
- Generated on every CI run
- Uploaded to Codecov
- Checked against 60% threshold
- Failed builds if below threshold

### Artifacts

Build artifacts are uploaded for:
- Binary builds (multiple OS/arch combinations)
- Test results
- Coverage reports

## Coverage Requirements

### Current Threshold

**Minimum coverage**: 60%

This is enforced in CI and will fail builds that don't meet the threshold.

### Checking Coverage Locally

```bash
# Generate coverage
make coverage

# View in browser
open coverage.html

# Check percentage
go tool cover -func=coverage.out | grep total
```

### Improving Coverage

Focus on:
1. Core Raft logic (election, replication, snapshots)
2. State machine operations
3. Error handling paths
4. Edge cases in cluster membership

## Best Practices

### 1. Test Independence

Each test should be independent and not rely on state from other tests.

```go
// Good: Each test creates its own cluster
func TestElection(t *testing.T) {
    cluster := testutil.NewTestCluster(t, config)
    defer cluster.Stop()
    // test logic
}

// Bad: Sharing state between tests
var sharedCluster *testutil.TestCluster
```

### 2. Timeout Management

Always set appropriate timeouts:

```go
// Good: Reasonable timeout
leader := cluster.WaitForLeader(5 * time.Second)

// Bad: No timeout or too long
leader := cluster.WaitForLeader(1 * time.Hour)
```

### 3. Cleanup

Use defer for cleanup to prevent resource leaks:

```go
cluster := testutil.NewTestCluster(t, config)
defer cluster.Stop()  // Always cleanup

server := api.NewServer(leader)
defer server.Stop()
```

### 4. Descriptive Failures

Provide context in assertions:

```go
// Good
testutil.AssertEqual(t, expected, actual, "leader term should match")

// Bad
testutil.AssertEqual(t, expected, actual, "")
```

### 5. Test Flakiness

Avoid flaky tests:
- Use deterministic timeouts
- Avoid hardcoded sleeps; use `WaitFor` patterns
- Handle eventual consistency properly
- Use race detector to catch concurrency issues

```go
// Good: Wait for condition
testutil.AssertEventually(t, func() bool {
    return cluster.GetLeader() != nil
}, 5*time.Second, "leader should be elected")

// Bad: Fixed sleep
time.Sleep(1 * time.Second)
if cluster.GetLeader() == nil {
    t.Fatal("no leader")
}
```

### 6. Parallel Tests

Use `t.Parallel()` for independent tests to speed up execution:

```go
func TestParallelSafe(t *testing.T) {
    t.Parallel()
    // test logic that doesn't share state
}
```

### 7. Skip Long Tests in Short Mode

```go
func TestLongRunning(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping long test in short mode")
    }
    // long-running test logic
}
```

## Troubleshooting

### Tests Hanging

- Check for missing cleanup (defer statements)
- Look for goroutine leaks
- Verify timeouts are set appropriately

### Flaky Tests

- Enable race detector: `go test -race`
- Check for timing assumptions
- Use proper synchronization primitives
- Review event ordering assumptions

### Coverage Issues

- Identify uncovered code: `go tool cover -html=coverage.out`
- Add tests for error paths
- Test edge cases
- Don't test generated code (*.pb.go)

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Raft Paper](https://raft.github.io/raft.pdf)
- [Jepsen Testing](https://jepsen.io/)
- [Project README](../README.md)
- [Development Plan](../DEVELOPMENT_PLAN.md)
