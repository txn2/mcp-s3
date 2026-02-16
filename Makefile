# mcp-s3 Makefile

.PHONY: all build test lint clean coverage security help tidy verify fmt lint-fix test-short \
       test-integration deadcode bench profile build-check install docs-serve docs-build \
       docker-build run version

# Variables
BINARY_NAME := mcp-s3
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d ' ' -f 3)
COVERAGE_FILE := coverage.out
COVERAGE_THRESHOLD := 80

# Directories
CMD_DIR := ./cmd/mcp-s3
BUILD_DIR := ./build
DIST_DIR := ./dist

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build
GOMOD := $(GO) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Linker flags (strip symbols for smaller binaries)
LDFLAGS := -ldflags "-s -w -X github.com/txn2/mcp-s3/internal/server.Version=$(VERSION)"

## all: Lint, test, and build
all: lint test build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## test: Run tests with race detection
test:
	$(GOTEST) -v -race -shuffle=on -count=1 ./...

## test-short: Run tests without race detection (faster)
test-short:
	$(GOTEST) -v ./...

## test-integration: Run integration tests
test-integration:
	$(GOTEST) -v -tags=integration ./...

## coverage: Generate coverage report with threshold enforcement
coverage:
	$(GOTEST) -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@COVERAGE=$$($(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$NF}' | sed 's/%//'); \
	echo "Coverage: $${COVERAGE}%"; \
	if [ $$(echo "$${COVERAGE} < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "FAIL: Coverage $${COVERAGE}% is below threshold $(COVERAGE_THRESHOLD)%"; \
		exit 1; \
	fi

## coverage-html: Generate HTML coverage report
coverage-html: coverage
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html

## lint: Run golangci-lint + go vet
lint:
	$(GOLINT) run --timeout=5m
	$(GO) vet ./...

## lint-fix: Run linter with auto-fix
lint-fix:
	$(GOLINT) run --fix --timeout=5m

## fmt: Format code
fmt:
	$(GO) fmt ./...
	goimports -w -local github.com/txn2/mcp-s3 .

## security: Run gosec + govulncheck
security:
	gosec ./...
	govulncheck ./...

## deadcode: Detect unreachable functions
deadcode:
	deadcode ./...

## bench: Run benchmarks with memory reporting
bench:
	$(GOTEST) -bench=. -benchmem -count=3 -run='^$$' ./... | tee bench.txt

## profile: Generate CPU and memory profiles
profile:
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof -run='^$$' ./...
	@echo "CPU profile: go tool pprof cpu.prof"
	@echo "Memory profile: go tool pprof mem.prof"

## build-check: Verify build and modules
build-check:
	$(GO) build ./...
	$(GO) mod verify

## tidy: Tidy and verify modules
tidy:
	$(GO) mod tidy
	$(GO) mod verify

## clean: Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html bench.txt cpu.prof mem.prof
	$(GO) clean -cache -testcache

## install: Install the binary
install: build
	$(GO) install $(LDFLAGS) $(CMD_DIR)

## verify: Run full verification suite
verify: tidy lint test coverage security deadcode build-check
	@echo "All verification checks passed."

## docker-build: Build Docker image
docker-build:
	docker build -t txn2/mcp-s3:$(VERSION) .
	docker tag txn2/mcp-s3:$(VERSION) txn2/mcp-s3:latest

## run: Run the server
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

## version: Show version
version:
	@echo "Version: $(VERSION)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Build Time: $(BUILD_TIME)"

## docs-serve: Serve documentation locally
docs-serve:
	python3 -m mkdocs serve

## docs-build: Build documentation
docs-build:
	python3 -m mkdocs build

## help: Show this help message
help:
	@echo "mcp-s3 Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              - Run lint, test, and build (default)"
	@echo "  build            - Build the binary"
	@echo "  test             - Run tests with race detection"
	@echo "  test-short       - Run tests without race detection"
	@echo "  test-integration - Run integration tests"
	@echo "  coverage         - Generate coverage report (threshold: $(COVERAGE_THRESHOLD)%)"
	@echo "  coverage-html    - Generate HTML coverage report"
	@echo "  lint             - Run golangci-lint + go vet"
	@echo "  lint-fix         - Run golangci-lint with auto-fix"
	@echo "  fmt              - Format code"
	@echo "  security         - Run gosec + govulncheck"
	@echo "  deadcode         - Detect unreachable functions"
	@echo "  bench            - Run benchmarks with memory reporting"
	@echo "  profile          - Generate CPU and memory profiles"
	@echo "  build-check      - Verify build and modules"
	@echo "  tidy             - Tidy and verify modules"
	@echo "  clean            - Remove build artifacts"
	@echo "  verify           - Run full verification suite"
	@echo "  help             - Show this help"
