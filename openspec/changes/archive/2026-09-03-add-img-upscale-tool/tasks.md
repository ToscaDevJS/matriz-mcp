# Tasks: Dedicated img_upscale Tool

## Phase 1: Test Suite & Failure Modes (TDD RED)
- [x] 1.1 Add test for `ToolUpscale` description cost marker (`COSTS MONEY`) in `internal/mcpserver/server_test.go` (RED).
- [x] 1.2 Add test `TestUpscale_EndToEnd` with thumbnail and sidecar assertion in `internal/mcpserver/tools_generative_test.go` (RED).
- [x] 1.3 Add test for budget exhaustion guard on `img_upscale` (RED).

## Phase 2: Implementation of img_upscale (TDD GREEN)
- [x] 2.1 Define `ToolUpscale`, `UpscaleIn`, `UpscaleOut` and register in `internal/mcpserver/server.go`.
- [x] 2.2 Implement `handleUpscale` in `internal/mcpserver/tools_generative.go` utilizing `CapabilityUpscale` and `ModelFinal`.
- [x] 2.3 Implement test helper `CallUpscale` in `internal/mcpserver/server.go`.
- [x] 2.4 Verify all tests pass with `go test -v -count=1 ./internal/mcpserver/...`.

## Phase 3: MCP Tool Registration & Schema Export
- [x] 3.1 Export JSON schema `~/.gemini/antigravity-cli/mcp/matriz/img_upscale.json`.
- [x] 3.2 Update `~/.gemini/antigravity-cli/mcp/matriz/instructions.md` to reference `img_upscale`.
- [x] 3.3 Rebuild binary `./bin/matriz-mcp`.
- [x] 3.4 Verify full repository test suite passes with `go test -v -count=1 ./...`.
