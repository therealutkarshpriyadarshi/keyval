#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
COVERAGE_THRESHOLD=60
COVERAGE_FILE="coverage.out"

echo -e "${YELLOW}Running KeyVal Test Suite${NC}"
echo "========================================"

# Function to print section header
print_header() {
    echo -e "\n${GREEN}>>> $1${NC}"
}

# Function to print error
print_error() {
    echo -e "${RED}ERROR: $1${NC}"
}

# Function to print success
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed"
    exit 1
fi

# Generate protobuf code if needed
print_header "Generating protobuf code"
if [ -d "proto" ] && [ -f "proto/raft.proto" ]; then
    if ! command -v protoc &> /dev/null; then
        print_error "protoc is not installed"
        exit 1
    fi

    export PATH=$PATH:$(go env GOPATH)/bin
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        proto/raft.proto

    print_success "Protobuf code generated"
else
    print_error "Proto files not found"
    exit 1
fi

# Download dependencies
print_header "Downloading dependencies"
go mod download
go mod tidy
print_success "Dependencies ready"

# Run linters
print_header "Running linters"
go vet ./...
gofmt -l . | tee /tmp/gofmt.out
if [ -s /tmp/gofmt.out ]; then
    print_error "Code is not formatted. Run 'make fmt'"
    exit 1
fi
print_success "Linting passed"

# Run unit tests
print_header "Running unit tests"
go test -v -race -short ./... | tee /tmp/test.out
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    print_error "Unit tests failed"
    exit 1
fi
print_success "Unit tests passed"

# Run integration tests
print_header "Running integration tests"
go test -v -tags=integration -timeout=10m ./test/integration/... | tee /tmp/integration.out
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    print_error "Integration tests failed"
    exit 1
fi
print_success "Integration tests passed"

# Run E2E tests
print_header "Running E2E tests"
go test -v -tags=e2e -timeout=15m ./test/e2e/... | tee /tmp/e2e.out
if [ ${PIPESTATUS[0]} -ne 0 ]; then
    print_error "E2E tests failed"
    exit 1
fi
print_success "E2E tests passed"

# Generate coverage report
print_header "Generating coverage report"
go test -race -coverprofile=$COVERAGE_FILE -covermode=atomic ./...

if [ -f $COVERAGE_FILE ]; then
    # Calculate coverage
    coverage=$(go tool cover -func=$COVERAGE_FILE | grep total | awk '{print $3}' | sed 's/%//')

    echo -e "Total coverage: ${GREEN}${coverage}%${NC}"

    # Check coverage threshold
    if (( $(echo "$coverage < $COVERAGE_THRESHOLD" | bc -l) )); then
        print_error "Coverage ${coverage}% is below threshold ${COVERAGE_THRESHOLD}%"
        exit 1
    fi

    # Generate HTML report
    go tool cover -html=$COVERAGE_FILE -o coverage.html
    print_success "Coverage report generated: coverage.html"
else
    print_error "Coverage file not generated"
    exit 1
fi

# Run benchmarks (optional)
if [ "$RUN_BENCHMARKS" = "true" ]; then
    print_header "Running benchmarks"
    go test -bench=. -benchmem -run=^$ ./test/bench/... | tee /tmp/bench.out
    print_success "Benchmarks completed"
fi

# Run chaos tests (optional, very time-consuming)
if [ "$RUN_CHAOS" = "true" ]; then
    print_header "Running chaos tests"
    go test -v -tags=chaos -timeout=30m ./test/chaos/... | tee /tmp/chaos.out
    if [ ${PIPESTATUS[0]} -ne 0 ]; then
        print_error "Chaos tests failed"
        exit 1
    fi
    print_success "Chaos tests passed"
fi

echo -e "\n${GREEN}========================================"
echo -e "All tests passed! ✓${NC}"
echo -e "========================================\n"
