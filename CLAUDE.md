# CLAUDE.md

This file provides guidance to Claude Code when working with this project.

## Project Overview

**mcp-s3** is a generic, open-source MCP (Model Context Protocol) server for Amazon S3 and S3-compatible object storage. It enables AI assistants to interact with object storage via the MCP protocol.

**Key Design Goals:**
- Composable: Can be used standalone OR imported as a library
- Generic: No domain-specific logic; suitable for any S3-compatible deployment
- Secure: Read-only by default with configurable limits

## CRITICAL - Factual Integrity (No Confabulation)

AI-generated prose (PR descriptions, commit messages, reviews, explanations) is held to the same verification standard as code. Unverified claims are as unacceptable as untested code.

1. **Never assert facts you haven't verified.** Before stating that a file contains X, a config is missing Y, or a system behaves in way Z — READ the file, CHECK the config, VERIFY the behavior. If you haven't looked, say "I haven't verified this" or say nothing.

2. **Every claim must be evidence-linked.** PR descriptions, commit messages, and work summaries may only include claims that are either: (a) directly visible in the diff, or (b) verified by reading a specific file (cite file:line). No exceptions.

3. **Never pad or embellish.** If you made two fixes, describe two fixes. Do not invent a third to make the work look more complete. Do not present hypotheses as confirmed diagnoses.

4. **Uncertainty must be explicit.** Use "I believe," "possibly," or "I haven't verified" when uncertain. Never upgrade a guess to a fact.

5. **When reviewing, verify claims against evidence.** Treat PR descriptions and commit messages as claims to be fact-checked, not trusted context.

6. **Omission over fabrication.** A gap stated honestly is better than a fabricated answer stated confidently. When in doubt, leave it out.

7. **Report a failing gate as a failure, in the first sentence.** A summary that lists the checks that passed and mentions the one that failed further down reads as a pass and is not one. If `make verify` exits non-zero, the report opens with that. Enumerating green sub-checks above a red overall result is a form of padding.

## Development Workflow

Work reaches the human in this order. The steps are not interchangeable, and none of them is optional.

1. **Acceptance criteria first.** State the observable Given/When/Then before writing code.
2. **Implement**, with tests that encode expected outputs.
3. **Adversarial review.** Spawn a `general-purpose` sub-agent with the prompt in `~/.claude/hooks/review-prompt-template.md` — verbatim, and do not write the review yourself. Address every finding: fix it, dispute it in writing in the commit body, or defer it with a TODO. Re-spawn on the new tree. Stop at `Verdict: CLEAN` or when only comment polish remains. Iteration cap: 1 round for typical changes, 2 only if round 1 found a substantive bug. Write the verdict to `.claude/.last-review.md`.
4. **`make verify`** — the LAST step, run on the exact tree being committed. It writes `.claude/.last-verify-passed`, which the pre-commit gate compares against the live diff. Verifying before the review means verifying a tree the review then changes.
5. **Human review.** A human reads and approves every line, then the commit happens. Commits are performed by a human, not by Claude.

Running `make verify` before the adversarial review inverts the order and wastes the run: any finding the review produces invalidates the sentinel it just wrote.

## Code Standards

1. **Idiomatic Go**: All code must follow idiomatic Go patterns and conventions. Use `gofmt`, follow Effective Go guidelines, and adhere to Go Code Review Comments.

2. **Test Coverage**: Project must maintain >=82% total unit test coverage (`COVERAGE_THRESHOLD` in the Makefile), and >=80% coverage of the lines a branch changes (`PATCH_COVERAGE_THRESHOLD`, mirroring the codecov patch check). Build mocks where necessary. Use table-driven tests where appropriate.
   - A total-coverage gate cannot see new untested code when the rest of the tree carries the average. The patch gate is what actually holds new code to a standard.

3. **Testing Definition**: When asked to "test" or "testing" the code, this means running `make verify`, which executes the full CI-equivalent suite:
   - **Tools-check (parity gate)** — verifies local `golangci-lint`, `gosec`, and `govulncheck` versions equal `GOLANGCI_LINT_VERSION`, `GOSEC_VERSION`, and `GOVULNCHECK_VERSION` in the Makefile. Those variables are the single source of truth: CI installs the same versions by reading them back with `make print-<VAR>`, rather than carrying hand-copied duplicates that drift. Drifting local tool versions are the most insidious parity gap: a local scanner that is newer than CI's can silently relax a rule CI's pinned build still enforces, so `make verify` goes green on a diff CI then rejects. `make verify` refuses to run until local matches CI. Override with `TOOLS_CHECK_STRICT=0` only with an explicit reason.
   - Module tidy + verify (`go mod tidy`, `go mod verify`)
   - Linting (`golangci-lint run` + `go vet ./...`) — cyclomatic complexity <=10, cognitive complexity <=15 in non-test code
   - Unit tests with race detection and shuffling (`go test -race -shuffle=on -count=1 ./...`)
   - Total coverage (hard gate, `COVERAGE_THRESHOLD`)
   - Patch coverage — changed lines vs the merge base with main (hard gate, `PATCH_COVERAGE_THRESHOLD`)
   - Security scanning (`gosec` + `govulncheck`, whose report is judged against `.govulncheck-allow.txt` by `scripts/govulncheck-gate.py`: an advisory with no patched release may be accepted with a written reason, and the gate fails when an accepted advisory gains a fix or stops being reported). CI runs this same `make security` target, so an accepted advisory cannot pass locally and fail in CI.
   - Dead code analysis
   - Build check (`go build ./...`)
   - All checks must pass locally before considering code "tested"

   **Why govulncheck is gated rather than raw**: govulncheck exits 3 for "your code calls a vulnerable symbol" whether or not a fix exists, and has no way to accept a finding. Five Go standard-library advisories once held `make verify` red on `main` for long enough that the suite stopped being run at all. A gate that cannot be satisfied is a gate that gets ignored.

4. **CRITICAL - Coverage Verification Before Completion**: Before declaring ANY implementation task complete:
   - Run `go test -coverprofile=coverage.out ./...` (note `./...`, not `./pkg/...` — covers `cmd/` too)
   - For EVERY new function or method added, run `go tool cover -func=coverage.out | grep <function_name>`
   - **If ANY new function shows less than 80% coverage (or 0.0%), you MUST add tests before declaring done**
   - This is BLOCKING — do not report the work complete until all new code has adequate test coverage

5. **CRITICAL - Integration Tests for Cross-Component Behavior**: Unit tests are NOT sufficient for behavior that only appears once components are assembled — MCP tool registration, middleware and interceptor chains, schema validation, context propagation. Before declaring such work complete:
   - **Write a test that exercises the real assembled system.** For this project that means registering the toolkit on a real `mcp.Server`, connecting an in-memory client (`mcp.NewInMemoryTransports`), and calling the tool through `CallTool` — not calling the handler function directly.
   - **A handler-level test proves the handler works. It does not prove the tool works.** The SDK validates arguments and structured output around the handler, so a handler can return a correct result that never reaches the client. Issue #141 was exactly this: every handler test passed while three tools returned a JSON-RPC error to real clients.
   - Assert the failure paths through the same wiring, not only the success path.

6. **Human Review Required**: A human must review and approve every line of code before it is committed. Therefore, commits are always performed by a human, not by Claude.

7. **Go Report Card**: The project MUST always maintain 100% across all categories on [Go Report Card](https://goreportcard.com/). This includes:
   - **gofmt**: All code must be formatted with `gofmt`
   - **go vet**: No issues from `go vet`
   - **gocyclo**: All functions must have cyclomatic complexity <=10 (non-test code)
   - **golint**: No lint issues (deprecated but still checked)
   - **ineffassign**: No ineffectual assignments
   - **license**: Valid license file present
   - **misspell**: No spelling errors in comments/strings

8. **Diagrams**: Use Mermaid for all diagrams. Never use ASCII art.

9. **Pinned Dependencies**: All external dependencies must be pinned for reproducibility and security:
   - GitHub Actions: pinned by commit SHA with a trailing version comment
   - Scanner versions (`golangci-lint`, `gosec`, `govulncheck`): pinned in the Makefile ONLY. CI reads each one back with `make print-<VAR>` rather than carrying a copy, and `make tools-check` holds local binaries to the same pins. Nothing enforces agreement between two hand-written copies, which is why there is only one.
   - Go toolchain: pinned by the `toolchain` directive in `go.mod`, which CI reads via `go-version-file: go.mod`
   - Go modules: pinned via `go.sum`
   - **Actions published as a family must move together.** `github/codeql-action/{init,autobuild,analyze,upload-sarif}` reject a version mismatch between sub-actions in the same run. Dependabot opens one PR per sub-action and does not always open one for every member, so a rollup must check every reference, not only the ones with PRs.

## Architecture

```
mcp-s3/
├── cmd/mcp-s3/main.go         # Standalone server entrypoint
├── pkg/                        # PUBLIC API (importable by other projects)
│   ├── client/                 # S3 client wrapper
│   │   ├── client.go           # Connection and S3 operations
│   │   └── config.go           # Configuration from env/struct
│   ├── tools/                  # MCP tool definitions
│   │   ├── toolkit.go          # NewToolkit() and RegisterAll()
│   │   ├── list_buckets.go     # s3_list_buckets tool
│   │   ├── list_objects.go     # s3_list_objects tool
│   │   ├── get_object.go       # s3_get_object tool
│   │   └── ...                 # Other tools
│   ├── extensions/             # Built-in extensions
│   │   ├── readonly.go         # Block write operations
│   │   ├── sizelimit.go        # Enforce size limits
│   │   └── ...                 # Other extensions
│   ├── integration/            # Extension interfaces for custom integrations
│   └── multiserver/            # Multi-account support
├── internal/server/            # Default server setup (private)
├── go.mod
├── LICENSE                     # Apache 2.0
└── README.md
```

## Key Dependencies

- `github.com/aws/aws-sdk-go-v2` - AWS SDK for Go v2
- `github.com/mark3labs/mcp-go` - MCP SDK for Go

## Building and Running

```bash
# Build
go build -o mcp-s3 ./cmd/mcp-s3

# Run with AWS credentials
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
./mcp-s3
```

## Testing with SeaweedFS

```bash
# Start local SeaweedFS with S3 API
docker run -d -p 8333:8333 chrislusf/seaweedfs server -s3

# Configure for local testing
export S3_ENDPOINT=http://localhost:8333
export S3_USE_PATH_STYLE=true
export AWS_ACCESS_KEY_ID=any
export AWS_SECRET_ACCESS_KEY=any
export AWS_REGION=us-east-1

./mcp-s3
```

## Composition Pattern

This package is designed to be imported by other MCP servers:

```go
import (
    "github.com/txn2/mcp-s3/pkg/client"
    "github.com/txn2/mcp-s3/pkg/tools"
)

// Create client
s3Client, _ := client.New(ctx, client.FromEnv())

// Create toolkit and register on your server
toolkit := tools.NewToolkit(s3Client)
toolkit.RegisterAll(yourMCPServer)
```

## MCP Tools

| Tool | Description |
|------|-------------|
| `s3_list_buckets` | List accessible S3 buckets |
| `s3_list_objects` | List objects with prefix/delimiter/pagination |
| `s3_get_object` | Retrieve object content |
| `s3_get_object_metadata` | Get metadata without content (HEAD) |
| `s3_put_object` | Upload object (blocked by default) |
| `s3_delete_object` | Delete object (blocked by default) |
| `s3_copy_object` | Copy object within/between buckets |
| `s3_presign_url` | Generate presigned GET/PUT URLs |
| `s3_list_connections` | List configured S3 connections |

## Configuration Reference

Environment variables:
- `AWS_REGION` - AWS region (default: us-east-1)
- `AWS_ACCESS_KEY_ID` - Access key
- `AWS_SECRET_ACCESS_KEY` - Secret key
- `AWS_SESSION_TOKEN` - Session token (optional)
- `AWS_PROFILE` - Profile name (optional)
- `S3_ENDPOINT` - Custom endpoint for S3-compatible storage
- `S3_USE_PATH_STYLE` - Use path-style URLs (required for most S3-compatible storage)
- `S3_TIMEOUT` - Operation timeout (default: 30s)
- `MCP_S3_EXT_READONLY` - Block write operations (default: true)
- `MCP_S3_EXT_SIZELIMIT` - Enforce size limits (default: true)
- `MCP_S3_MAX_GET_SIZE` - Max bytes for GET (default: 10MB)
- `MCP_S3_MAX_PUT_SIZE` - Max bytes for PUT (default: 100MB)

## Verification (AI-Verified Development)

Run the full verification suite as the LAST step before every commit, after the
adversarial review (see Development Workflow above):
```
make verify
```

It writes `.claude/.last-verify-passed`, the short SHA-256 of the working-tree
diff it passed against. The pre-commit gate compares that hash to the live diff,
so a verify run only clears the tree it actually ran on. The hash computation
must stay byte-identical to `compute_diff_hash()` in `~/.claude/hooks/review-gate.sh`.

Individual checks (all must pass):
```
make tools-check     # Local tool versions == CI-pinned versions (runs first)
make lint            # golangci-lint (25 linters) + go vet
make test            # go test -race -shuffle=on ./...
make coverage        # Total coverage (threshold: 82%)
make patch-coverage  # Coverage of changed lines vs main (threshold: 80%)
make security        # gosec + govulncheck, gated by .govulncheck-allow.txt
make deadcode        # deadcode (unreachable functions)
make build-check     # go build + go mod verify
```

Performance diagnostics (not part of verify, use when investigating):
```
make bench           # Run benchmarks with memory allocation reporting
make profile         # Generate CPU and memory profiles for pprof
```

## Code Quality Thresholds

- Total test coverage: >=82%
- Patch coverage (changed lines vs main): >=80%
- Cyclomatic complexity: <=10 per function (non-test code)
- Cognitive complexity: <=15 per function (non-test code)
- Line length: <=140 characters

Table-driven tests are exempt from both complexity limits: their branching lives
in the case table, not in logic a reader has to hold.

## Go Code Standards (AI-Verified)

1. **Error handling**: Always wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
2. **Naming**: Follow Go conventions. MixedCaps, not underscores. Acronyms are all-caps (HTTP, URL, ID).
3. **Interfaces**: Accept interfaces, return structs. Define interfaces at the consumer, not the provider.
4. **Context**: First parameter when needed. Never store in structs.
5. **Concurrency**: Use channels for communication, mutexes for state. Always run tests with `-race`.
6. **Dependencies**: Use `internal/` for code that shouldn't be imported. Minimize third-party dependencies.
7. **Testing**: Table-driven tests. Property-based tests for pure functions.

## AI Verification Requirements

When AI (Claude Code or similar) contributes code, these additional checks apply.

1. **No Tautological Tests**: Tests must verify behavior, not struct field assignment. A test that sets `x.Field = "value"` then asserts `x.Field == "value"` tests the Go compiler, not the application. Delete such tests on sight. A test that would still pass if the production code returned a hardcoded value is not testing the production code.

2. **Integration Tests for Multi-Component Features**: See Code Standards #5. Unit tests alone are insufficient for anything that crosses the SDK boundary; require a test that wires up a real `mcp.Server` with an in-memory transport and calls the tool.

3. **Test the Failure Path Through the Real Wiring**: A suite that only validates success bodies cannot see an error result that never reaches the client. Every tool that can fail needs a test that makes it fail — through the server, not the handler.

4. **Dead Code Audit**: Run `make deadcode` before submitting. Functions reported as dead should be deleted or moved to test files. Public API functions are false positives here (this package is a library) and may be ignored with justification.

5. **No Vaporware**: Every package under `pkg/` must be imported by at least one non-test file. Every interface with a noop implementation must also have a real implementation. Do not create packages or interfaces "for future use" — code not wired into the running application is dead code regardless of whether it has its own unit tests.
   - **The Noop Loophole**: a noop implementation satisfies compile checks, passes tests (returns nil), gets imported (not dead), and wires into the server — yet does nothing. It is the most insidious form of vaporware because every automated gate reports green.

6. **Dependency-First Verification**: Before implementing anything that depends on an external capability (an SDK behavior, an S3 API operation), VERIFY the dependency actually supports it — read the vendored source in `$(go env GOMODCACHE)`, do not infer from the name. If the upstream library lacks the needed behavior, surface that gap IMMEDIATELY rather than building scaffolding around a capability that does not exist.

7. **No Hallucinated Imports**: verify every dependency exists in the Go module ecosystem.

8. **Acceptance Criteria First**: do not write code without Given/When/Then criteria. The criteria must describe user-visible behavior, not internal implementation details.

9. **Explain Non-Obvious Decisions**: comment WHY, not WHAT.

10. **Upstream Behavior Belongs in a Comment**: when code exists to work around or accommodate a dependency's behavior, the comment must name the behavior and where it lives, so a future reader can check whether it still holds after a version bump.
