# Tasks: Gemini Concurrent Multi-Draft Generation

## Phase 1: Test Suite & Failure Modes (TDD RED)
- [x] 1.1 Add unit tests for parallel fan-out, partial failure tolerance, and seed offset calculation in `internal/providers/gemini/gemini_test.go` (RED).
- [x] 1.2 Verify tests fail or assert expected concurrency behavior.

## Phase 2: Implementation of Parallel Fan-Out (TDD GREEN)
- [x] 2.1 Implement concurrent worker fan-out in `internal/providers/gemini/gemini.go` using bounded goroutines.
- [x] 2.2 Implement per-worker seed offsetting (`seed + i`).
- [x] 2.3 Implement thread-safe image aggregation and best-effort partial error handling.
- [x] 2.4 Verify all tests pass with `go test -v -count=1 ./internal/providers/...`.

## Phase 3: Verification & Integration Test
- [x] 3.1 Update `openspec/specs/provider-budget/spec.md`.
- [x] 3.2 Update `PENDING.md` (resolve item #5).
- [x] 3.3 Verify full test suite passes with `go test -v -count=1 ./...`.
