# Tasks: Matriz Implementation (v0.1.0)

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~2400 lines (across 6 PR units) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR-0 → PR-1 → PR-2 → PR-3 → PR-4 → PR-5 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 0 | Scaffold & Core Types | PR-0 | `go test ./internal/core -run TestResolveRef` | `go build ./...` | `internal/core/`, `internal/config/` |
| 1 | Deterministic Pipeline | PR-1 | `go test ./internal/core -run "TestResize|TestAdjust|TestExportWeb|TestSidecar"` | `go test ./internal/core` | `internal/core/transform.go`, `export.go` |
| 2 | Providers & Budget Guard | PR-2 | `go test ./internal/budget ./internal/providers/...` | `go test ./...` | `internal/providers/`, `internal/budget/` |
| 3 | MCP Server & Tools | PR-3 | `go test ./internal/mcpserver/...` | MCP Inspector | `cmd/matriz-mcp/`, `internal/mcpserver/` |
| 4 | Manifest Resource | PR-4 | `go test ./internal/manifest/...` | MCP Inspector | `internal/manifest/` |
| 5 | Curation TUI | PR-5 | `go test ./internal/tui/...` | `go run ./cmd/matriz-tui` | `cmd/matriz-tui/`, `internal/tui/` |

## Phase 0: Scaffold & Core Types (PR-0)
- [x] 0.1 RED: Write `T-01` spike and `T-02` tests for `ResolveRef` and path traversal defense in `internal/core/paths_test.go`
- [x] 0.2 GREEN: Initialize `go.mod`, add types in `internal/core/types.go`, errors in `internal/core/errors.go`, `ResolveRef` in `internal/core/paths.go`, and config in `internal/config/config.go`
- [x] 0.3 REFACTOR: Verify `go test ./...` and `go vet ./...` pass cleanly

## Phase 1: Deterministic Pipeline (PR-1)
- [x] 1.1 RED: Write golden file tests (`T-03`, `T-04`) and unit tests (`T-05`, `T-06`, `T-07`) in `internal/core/`
- [x] 1.2 GREEN: Implement `transform.go`, `encode.go`, `export.go`, and `sidecar.go`
- [x] 1.3 REFACTOR: Generate reference golden fixtures with `-update` and verify byte-for-byte passes

## Phase 2: Providers, Gemini & Budget Guard (PR-2)
- [x] 2.1 RED: Write `T-08`, `T-09`, `T-10`, `T-10b`, `T-10c`, `T-10d` tests in `internal/budget/` and `internal/providers/`
- [x] 2.2 GREEN: Implement `Provider` interface, `registry.go`, `fakeProvider`, `gemini.go`, `pricing.go`, and `Guard`
- [x] 2.3 REFACTOR: Verify offline determinism and budget ceiling enforcement

## Phase 3: MCP Server (PR-3)
- [x] 3.1 RED: Write `T-11` to `T-15` tool execution and thumbnail tests in `internal/mcpserver/`
- [x] 3.2 GREEN: Implement `cmd/matriz-mcp/main.go`, `tools_*.go`, `thumbnail.go`, and `errors.go`
- [x] 3.3 REFACTOR: Verify thumbnail sizes (<= 512px) and tool annotations

## Phase 4: Site Manifest Resource (PR-4)
- [ ] 4.1 RED: Write `T-16`, `T-17`, `T-18` tests for manifest schema, scanner, and MCP resource
- [ ] 4.2 GREEN: Implement `internal/manifest/manifest.go`, `scan.go`, and resource registration
- [ ] 4.3 REFACTOR: Verify round-trip manifest sync from filesystem

## Phase 5: Curation TUI (PR-5)
- [ ] 5.1 RED: Write `T-19` Bubbletea model unit test in `internal/tui/`
- [ ] 5.2 GREEN: Implement `internal/tui/model.go`, `keys.go`, `internal/preview/html.go`, and `cmd/matriz-tui/main.go`
- [ ] 5.3 REFACTOR: Verify full test suite `go test ./...` and `go vet ./...` across entire workspace
