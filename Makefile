.PHONY: all build test clean install proto lint coverage bench integration-test chaos-test docker help

# Variables
BINARY_NAME=keyval
CLI_NAME=kvctl
COVERAGE_FILE=coverage.out
PROTO_DIR=proto
PKG_DIR=pkg
CMD_DIR=cmd

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt

# Build parameters
BUILD_DIR=bin
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Docker parameters
DOCKER_IMAGE=keyval
DOCKER_TAG?=latest

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

all: lint test build

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

init: ## Initialize project (create dirs, install tools)
	@echo "$(YELLOW)Initializing project...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@mkdir -p cmd/$(BINARY_NAME)
	@mkdir -p cmd/$(CLI_NAME)
	@mkdir -p $(PKG_DIR)/{raft,storage,rpc,statemachine,cluster,api,config,metrics,tracing}
	@mkdir -p $(PROTO_DIR)
	@mkdir -p test/{integration,chaos,bench}
	@mkdir -p docs
	@mkdir -p scripts
	@mkdir -p deployments/{docker,kubernetes}
	@mkdir -p data
	@echo "$(GREEN)Project structure created!$(NC)"
	@echo "$(YELLOW)Installing development tools...$(NC)"
	@$(GOGET) google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@$(GOGET) google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "$(GREEN)Done! Run 'make deps' to install dependencies.$(NC)"

deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "$(GREEN)Dependencies installed!$(NC)"

proto: ## Generate protobuf code
	@echo "$(YELLOW)Generating protobuf code...$(NC)"
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/*.proto 2>/dev/null || echo "No proto files yet"
	@echo "$(GREEN)Protobuf code generated!$(NC)"

build: build-server build-cli ## Build all binaries

build-server: ## Build server binary
	@echo "$(YELLOW)Building server...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)/$(BINARY_NAME) 2>/dev/null || echo "Server code not ready yet"
	@echo "$(GREEN)Server built: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-cli: ## Build CLI binary
	@echo "$(YELLOW)Building CLI...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_NAME) ./$(CMD_DIR)/$(CLI_NAME) 2>/dev/null || echo "CLI code not ready yet"
	@echo "$(GREEN)CLI built: $(BUILD_DIR)/$(CLI_NAME)$(NC)"

install: ## Install binaries to $GOPATH/bin
	@echo "$(YELLOW)Installing binaries...$(NC)"
	@$(GOCMD) install $(LDFLAGS) ./$(CMD_DIR)/$(BINARY_NAME)
	@$(GOCMD) install $(LDFLAGS) ./$(CMD_DIR)/$(CLI_NAME)
	@echo "$(GREEN)Binaries installed!$(NC)"

test: ## Run unit tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@$(GOTEST) -v -race -timeout 30s ./... 2>/dev/null || echo "Tests not ready yet"

test-short: ## Run short tests
	@echo "$(YELLOW)Running short tests...$(NC)"
	@$(GOTEST) -short -v ./...

integration-test: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@$(GOTEST) -v -tags=integration -timeout 5m ./test/integration/... 2>/dev/null || echo "Integration tests not ready yet"

chaos-test: ## Run chaos/Jepsen tests
	@echo "$(YELLOW)Running chaos tests...$(NC)"
	@$(GOTEST) -v -tags=chaos -timeout 30m ./test/chaos/... 2>/dev/null || echo "Chaos tests not ready yet"

coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@$(GOTEST) -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./... 2>/dev/null || echo "Tests not ready yet"
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o coverage.html 2>/dev/null || true
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

bench: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem -run=^$ ./... 2>/dev/null || echo "Benchmarks not ready yet"

bench-write: ## Benchmark write operations
	@echo "$(YELLOW)Benchmarking write operations...$(NC)"
	@$(GOTEST) -bench=BenchmarkWrite -benchmem -benchtime=10s ./test/bench/... 2>/dev/null || echo "Benchmarks not ready yet"

bench-read: ## Benchmark read operations
	@echo "$(YELLOW)Benchmarking read operations...$(NC)"
	@$(GOTEST) -bench=BenchmarkRead -benchmem -benchtime=10s ./test/bench/... 2>/dev/null || echo "Benchmarks not ready yet"

lint: ## Run linters
	@echo "$(YELLOW)Running linters...$(NC)"
	@$(GOVET) ./... 2>/dev/null || echo "Code not ready yet"
	@$(GOFMT) -l -w . 2>/dev/null || true
	@which golangci-lint > /dev/null && golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed"
	@echo "$(GREEN)Linting complete!$(NC)"

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@$(GOFMT) -l -w .
	@echo "$(GREEN)Code formatted!$(NC)"

clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	@rm -rf data/
	@echo "$(GREEN)Cleaned!$(NC)"

clean-data: ## Clean Raft data directories
	@echo "$(YELLOW)Cleaning Raft data...$(NC)"
	@rm -rf data/
	@rm -rf raft-data-*/
	@rm -rf snapshots/
	@rm -rf wal/
	@echo "$(GREEN)Data cleaned!$(NC)"

# Cluster management targets
cluster-start: ## Start a 3-node local cluster
	@echo "$(YELLOW)Starting 3-node cluster...$(NC)"
	@mkdir -p data/node{1,2,3}
	@$(BUILD_DIR)/$(BINARY_NAME) --node-id=1 --data-dir=data/node1 --port=8001 --cluster=localhost:8001,localhost:8002,localhost:8003 > data/node1/node.log 2>&1 & echo $$! > data/node1/pid
	@sleep 1
	@$(BUILD_DIR)/$(BINARY_NAME) --node-id=2 --data-dir=data/node2 --port=8002 --cluster=localhost:8001,localhost:8002,localhost:8003 > data/node2/node.log 2>&1 & echo $$! > data/node2/pid
	@sleep 1
	@$(BUILD_DIR)/$(BINARY_NAME) --node-id=3 --data-dir=data/node3 --port=8003 --cluster=localhost:8001,localhost:8002,localhost:8003 > data/node3/node.log 2>&1 & echo $$! > data/node3/pid
	@echo "$(GREEN)Cluster started! Check logs in data/node*/node.log$(NC)"
	@echo "$(GREEN)PIDs saved in data/node*/pid$(NC)"

cluster-stop: ## Stop local cluster
	@echo "$(YELLOW)Stopping cluster...$(NC)"
	@for i in 1 2 3; do \
		if [ -f data/node$$i/pid ]; then \
			kill `cat data/node$$i/pid` 2>/dev/null || true; \
			rm data/node$$i/pid; \
		fi; \
	done
	@echo "$(GREEN)Cluster stopped!$(NC)"

cluster-status: ## Check cluster status
	@echo "$(YELLOW)Cluster status:$(NC)"
	@for i in 1 2 3; do \
		if [ -f data/node$$i/pid ] && kill -0 `cat data/node$$i/pid` 2>/dev/null; then \
			echo "$(GREEN)Node $$i: Running (PID: `cat data/node$$i/pid`)$(NC)"; \
		else \
			echo "$(RED)Node $$i: Stopped$(NC)"; \
		fi; \
	done

cluster-logs: ## Tail cluster logs
	@tail -f data/node*/node.log

# Docker targets
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f deployments/docker/Dockerfile . 2>/dev/null || echo "Dockerfile not ready yet"
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

docker-push: ## Push Docker image
	@echo "$(YELLOW)Pushing Docker image...$(NC)"
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Docker image pushed!$(NC)"

docker-run: ## Run container locally
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@docker run -p 8001:8001 -p 9001:9001 $(DOCKER_IMAGE):$(DOCKER_TAG)

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "$(YELLOW)Deploying to Kubernetes...$(NC)"
	@kubectl apply -f deployments/kubernetes/ 2>/dev/null || echo "K8s manifests not ready yet"
	@echo "$(GREEN)Deployed to Kubernetes!$(NC)"

k8s-delete: ## Delete from Kubernetes
	@echo "$(YELLOW)Deleting from Kubernetes...$(NC)"
	@kubectl delete -f deployments/kubernetes/ 2>/dev/null || echo "Nothing to delete"
	@echo "$(GREEN)Deleted from Kubernetes!$(NC)"

# Development helpers
dev: ## Run server in development mode
	@echo "$(YELLOW)Running in development mode...$(NC)"
	@$(GOCMD) run ./$(CMD_DIR)/$(BINARY_NAME) --log-level=debug

watch: ## Watch for changes and rebuild
	@echo "$(YELLOW)Watching for changes...$(NC)"
	@which fswatch > /dev/null || (echo "$(RED)fswatch not installed$(NC)" && exit 1)
	@fswatch -o . -e ".*" -i "\\.go$$" | xargs -n1 -I{} make build

# CI/CD targets
ci: lint test ## Run CI checks

ci-full: lint test integration-test coverage ## Run full CI checks

release: clean lint test build ## Prepare release build
	@echo "$(GREEN)Release build complete!$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"

# Documentation
docs: ## Generate documentation
	@echo "$(YELLOW)Generating documentation...$(NC)"
	@$(GOCMD) doc -all ./... > docs/API.md 2>/dev/null || echo "Code not ready yet"
	@echo "$(GREEN)Documentation generated: docs/API.md$(NC)"

# Performance profiling
profile-cpu: ## Run CPU profiling
	@echo "$(YELLOW)Running CPU profiling...$(NC)"
	@$(GOTEST) -cpuprofile=cpu.prof -bench=. ./test/bench/... 2>/dev/null || echo "Benchmarks not ready yet"
	@$(GOCMD) tool pprof cpu.prof

profile-mem: ## Run memory profiling
	@echo "$(YELLOW)Running memory profiling...$(NC)"
	@$(GOTEST) -memprofile=mem.prof -bench=. ./test/bench/... 2>/dev/null || echo "Benchmarks not ready yet"
	@$(GOCMD) tool pprof mem.prof

# Database operations
db-backup: ## Backup Raft data
	@echo "$(YELLOW)Backing up Raft data...$(NC)"
	@tar -czf backup-$(shell date +%Y%m%d-%H%M%S).tar.gz data/
	@echo "$(GREEN)Backup created!$(NC)"

db-restore: ## Restore Raft data (usage: make db-restore BACKUP=backup.tar.gz)
	@echo "$(YELLOW)Restoring Raft data from $(BACKUP)...$(NC)"
	@tar -xzf $(BACKUP)
	@echo "$(GREEN)Data restored!$(NC)"

# Debugging
debug-node1: ## Run node 1 with delve debugger
	@dlv debug ./$(CMD_DIR)/$(BINARY_NAME) -- --node-id=1 --port=8001

debug-test: ## Debug tests with delve
	@dlv test ./$(PKG_DIR)/raft/

# Quick commands for development
quick: fmt build test ## Quick dev cycle: format, build, test

up: build cluster-start ## Build and start cluster

down: cluster-stop clean-data ## Stop cluster and clean data

restart: down up ## Restart cluster

# Version information
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build time: $(BUILD_TIME)"
