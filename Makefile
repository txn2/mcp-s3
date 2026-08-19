# mcp-s3 Makefile

.PHONY: all build test lint clean coverage security help tidy verify fmt lint-fix test-short \
       test-integration deadcode bench profile build-check install docs-serve docs-build \
       docker-build run version tools-check patch-coverage print-%

# Variables
BINARY_NAME := mcp-s3
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d ' ' -f 3)
COVERAGE_FILE := coverage.out

# Total coverage across all packages, and the separate gate on the lines this
# branch changes. A branch can add untested code and still clear a total
# threshold, because the existing well-covered code carries the average; the
# patch gate is what actually holds new code to a standard. PATCH mirrors the
# codecov patch check so local and CI agree.
COVERAGE_THRESHOLD := 82
PATCH_COVERAGE_THRESHOLD := 80

# Tool versions. This block is the single source of truth: .github/workflows/ci.yml
# installs these exact versions by reading them back out with `make print-<VAR>`,
# and tools-check refuses to run verify when a local tool drifts from them.
#
# They are not mirrored into the workflow by hand. A hand-copied pin drifts: CI
# previously ran golang/govulncheck-action, which installs govulncheck@latest and
# accepts no version input, so there was no CI pin to mirror at all while
# tools-check was pinning developers to one.
#
# A local scanner that is NEWER than CI's is not the safe direction it sounds
# like: a rule the newer one relaxed is still enforced by CI's pinned build, so
# `make verify` goes green on a diff CI then rejects. This bit the sister
# project mcp-data-platform in PR #377, where a local gosec silently dropped an
# SSRF taint rule that CI still ran. Discipline alone was not enough there;
# neither is it here.
GOLANGCI_LINT_VERSION := v2.11.4
GOSEC_VERSION := v2.28.0
GOVULNCHECK_VERSION := v1.1.4

# govulncheck's JSON report, judged by scripts/govulncheck-gate.py.
GOVULN_REPORT := build/govulncheck-report.json

# Records the working-tree diff hash that `verify` last passed against, so a
# pre-commit gate can tell "verify is green on THIS tree" from "verify is green
# on some other tree". Written only after every check succeeds; gitignored.
VERIFY_SENTINEL := .claude/.last-verify-passed

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

## patch-coverage: Coverage of the lines this branch changed (mirrors codecov patch)
##
## Reads the profile `coverage` wrote. A total-coverage gate cannot see new
## untested code when the rest of the tree carries the average, which is the
## exact hole codecov's patch check exists to close.
patch-coverage:
	@echo "Checking patch coverage..."
	@PATCH_COVERAGE_THRESHOLD=$(PATCH_COVERAGE_THRESHOLD) COVERAGE_FILE=$(COVERAGE_FILE) ./scripts/patch-coverage.sh

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
	@echo "Running gosec..."
	gosec -quiet ./...
	@echo "Running govulncheck..."
	@# govulncheck offers no way to accept a finding: a called vulnerability
	@# fails the run whether or not a fixed version exists. That leaves one bad
	@# option — an advisory with no patched release turns every build red until
	@# upstream ships, so the whole gate gets ignored or switched off. This repo
	@# learned that the hard way: five Go stdlib advisories held `make verify`
	@# red on main, and the suite simply stopped being run.
	@#
	@# So the exit status is not the verdict here. Both 0 and 3 mean "the scan
	@# ran"; anything else means it could not run and is a hard error. The
	@# verdict comes from scripts/govulncheck-gate.py judging the report against
	@# .govulncheck-allow.txt, where an accepted advisory carries a written
	@# reason and expires the moment a fix ships or it stops being reported.
	@mkdir -p $(dir $(GOVULN_REPORT))
	@# stderr goes to a file rather than /dev/null: when govulncheck cannot run
	@# at all (build error, unreachable vuln DB) the reason is the only useful
	@# thing it produced, and discarding it leaves just an exit code to act on.
	@govulncheck -format json ./... > $(GOVULN_REPORT) 2>$(GOVULN_REPORT).err; \
	status=$$?; \
	if [ $$status -ne 0 ] && [ $$status -ne 3 ]; then \
		echo "ERROR: govulncheck failed to run (exit $$status)"; \
		cat $(GOVULN_REPORT).err; \
		rm -f $(GOVULN_REPORT) $(GOVULN_REPORT).err; \
		exit 1; \
	fi; \
	rm -f $(GOVULN_REPORT).err
	@python3 scripts/govulncheck-gate.py $(GOVULN_REPORT); \
	status=$$?; \
	rm -f $(GOVULN_REPORT); \
	exit $$status

## print-%: Print the value of any Makefile variable (used by CI to read the tool pins)
##
## CI installs scanners with `go install ...@$(make print-GOSEC_VERSION)` so the
## pins have exactly one home. Duplicating them into the workflow is how they
## drift apart without anyone noticing.
print-%:
	@echo "$($*)"

## tools-check: Fail when a local tool is missing or drifts from the CI-pinned version
##
## `make verify` claims to be CI-equivalent. That claim is only true while the
## local scanners are the same builds CI runs, so this gate runs first and
## refuses the suite otherwise — see the GOSEC_VERSION comment above for the
## incident that made a version pin necessary rather than advisory.
##
## Override with TOOLS_CHECK_STRICT=0 only with an explicit reason.
##
## Each arm reads the version from the binary's module metadata first, and falls
## back to parsing the tool's own version output when that is absent — which is
## the normal case for Homebrew, distro, and build-from-source installs.
##
## The govulncheck fallback anchors to the "Scanner: govulncheck@" line. Its
## -version output leads with the Go toolchain ("Go: go1.26.6"), so a bare
## version-shaped match returns the toolchain and hands every developer a false
## mismatch they cannot clear.
tools-check:
	@echo "Checking required tools (presence AND pinned versions)..."
	@missing=""; mismatch=""; \
	if ! which golangci-lint > /dev/null 2>&1; then \
		missing="$$missing  golangci-lint: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)\n"; \
	else \
		v=$$(go version -m $$(which golangci-lint) 2>/dev/null | awk '$$1=="mod" && $$2 ~ /golangci-lint/ {print $$3}'); \
		if [ -z "$$v" ] || [ "$$v" = "(devel)" ]; then \
			v=$$(golangci-lint version 2>&1 | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
			case "$$v" in v*) ;; *) v="v$$v";; esac; \
		fi; \
		if [ "$$v" != "$(GOLANGCI_LINT_VERSION)" ]; then \
			mismatch="$$mismatch  golangci-lint: have $$v, want $(GOLANGCI_LINT_VERSION) — go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)\n"; \
		fi; \
	fi; \
	if ! which gosec > /dev/null 2>&1; then \
		missing="$$missing  gosec: go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)\n"; \
	else \
		v=$$(go version -m $$(which gosec) 2>/dev/null | awk '$$1=="mod" && $$2 ~ /gosec/ {print $$3}'); \
		if [ -z "$$v" ] || [ "$$v" = "(devel)" ]; then \
			v=$$(gosec --version 2>&1 | grep -oE 'Version: v?[0-9]+\.[0-9]+\.[0-9]+' | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
			case "$$v" in v*) ;; *) v="v$$v";; esac; \
		fi; \
		if [ "$$v" != "$(GOSEC_VERSION)" ]; then \
			mismatch="$$mismatch  gosec: have $$v, want $(GOSEC_VERSION) — go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)\n"; \
		fi; \
	fi; \
	if ! which govulncheck > /dev/null 2>&1; then \
		missing="$$missing  govulncheck: go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"; \
	else \
		v=$$(go version -m $$(which govulncheck) 2>/dev/null | awk '$$1=="mod" && $$2=="golang.org/x/vuln" {print $$3}'); \
		if [ -z "$$v" ] || [ "$$v" = "(devel)" ]; then \
			v=$$(govulncheck -version 2>&1 | sed -n 's/^Scanner: govulncheck@//p' | head -1); \
		fi; \
		if [ -z "$$v" ] || [ "$$v" = "(devel)" ]; then v="unknown"; fi; \
		if [ "$$v" != "$(GOVULNCHECK_VERSION)" ]; then \
			mismatch="$$mismatch  govulncheck: have $$v, want $(GOVULNCHECK_VERSION) — go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)\n"; \
		fi; \
	fi; \
	which deadcode > /dev/null 2>&1 || missing="$$missing  deadcode: go install golang.org/x/tools/cmd/deadcode@latest\n"; \
	which python3 > /dev/null 2>&1  || missing="$$missing  python3: required by scripts/govulncheck-gate.py\n"; \
	if [ -n "$$missing" ]; then \
		echo ""; \
		echo "FAIL: Missing required tools:"; \
		printf '%b' "$$missing"; \
		echo ""; \
		echo "Install all missing tools before running make verify."; \
		exit 1; \
	fi; \
	if [ -n "$$mismatch" ]; then \
		echo ""; \
		echo "FAIL: Tool version mismatch (local differs from CI-pinned)."; \
		echo "Local versions that drift from CI's create silent parity gaps:"; \
		echo "make verify can pass while CI rejects the same diff."; \
		echo ""; \
		printf '%b' "$$mismatch"; \
		echo ""; \
		echo "Pin local tools to the CI versions before running make verify."; \
		echo "(Override with TOOLS_CHECK_STRICT=0 only if you know what you are doing.)"; \
		if [ "$(TOOLS_CHECK_STRICT)" != "0" ]; then exit 1; fi; \
		echo "WARN: proceeding with mismatched tool versions (TOOLS_CHECK_STRICT=0)."; \
	else \
		echo "All required tools found at pinned CI versions."; \
	fi

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

## verify: Run the CI-equivalent per-commit suite
##
## tools-check runs FIRST and on its own: every check after it is only
## CI-equivalent while the local tools match CI's pinned versions, so there is
## no point running the suite before that is established.
verify: tools-check tidy lint test coverage patch-coverage security deadcode build-check
	@echo ""
	@echo "=== All checks passed ==="
	@# Write the gate sentinel: the short SHA-256 of the working-tree diff
	@# (staged + unstaged) at the moment verify completed. The pre-commit
	@# review gate (~/.claude/hooks/review-gate.sh) compares this hash to the
	@# live diff at commit time — if they match, this verify run is proof that
	@# CI-equivalent checks passed on the exact code being committed, rather
	@# than on some earlier tree that has since moved.
	@#
	@# Hash computation MUST stay byte-identical to compute_diff_hash() in
	@# review-gate.sh, otherwise the gate rejects every commit.
	@mkdir -p .claude
	@{ git diff --cached HEAD 2>/dev/null; git diff 2>/dev/null; } \
		| shasum -a 256 | cut -c1-16 > $(VERIFY_SENTINEL)
	@echo "Wrote $(VERIFY_SENTINEL) (gate sentinel)"

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
