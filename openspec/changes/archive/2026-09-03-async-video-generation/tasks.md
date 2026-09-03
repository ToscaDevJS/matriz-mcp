# Tasks: Asynchronous Video Generation (Text-to-Video & Image-to-Video)

## PR-0: Core Capabilities & Two-Phase Budget Guard (TDD)
- [x] 0.1 Add video capabilities (`CapabilityVideoDraft`, `CapabilityVideoFinal`, `CapabilityImageToVideo`, `CapabilityTextToVideo`) in `internal/providers/provider.go`.
- [x] 0.2 Write unit tests in `internal/budget/budget_test.go` for ticket-based two-phase reservations: `Reserve -> ticketID`, `Commit(ticketID, actual)`, and `Release(ticketID)` (RED).
- [x] 0.3 Implement ticket reservation, concurrent lock tracking, and release in `internal/budget/budget.go` (GREEN).
- [x] 0.4 Refactor `internal/budget` to ensure 100% test coverage and backwards compatibility with static image tools.

## PR-1: VideoProvider Abstraction & Fake Implementation (TDD)
- [x] 1.1 Define `VideoProvider`, `VideoRequest`, `VideoJob`, and `VideoResult` in `internal/providers/video.go`.
- [x] 1.2 Write unit tests in `internal/providers/provider_test.go` asserting `VideoProvider` registration in `Registry` and capabilities discovery (RED).
- [x] 1.3 Implement `VideoProvider` on `internal/providers/fake/fake.go` with deterministic mock jobs and timing simulation (GREEN).
- [x] 1.4 Write helper `BuildVideoSidecar` in `internal/providers/video.go` enforcing `matriz.sidecar/v1` and `derived_from` provenance.

## PR-2: Google GenAI Video Provider (Veo 3.1 & Gemini Omni Flash)
- [x] 2.1 Add video model configuration in `internal/config/config.go` (`MATRIZ_MODEL_VIDEO_DRAFT`, `MATRIZ_MODEL_VIDEO_FINAL`).
- [x] 2.2 Implement `VideoProvider` interface on `internal/providers/gemini/gemini.go` using `client.Models.GenerateVideos` and `client.Operations.GetVideosOperation`.
- [x] 2.3 Implement error translation mapping Google safety blocks (`HARM_CATEGORY_*`) and quotas (`RESOURCE_EXHAUSTED`) to clean domain errors.

## PR-3: Job Engine & MCP Video Tools (TDD)
- [x] 3.1 Create `internal/jobs/engine.go` with thread-safe job tracking and smart-wait completion signaling.
- [x] 3.2 Write unit tests for `ToolVideoGenerate`, `ToolVideoStatus`, and `ToolVideoCancel` in `internal/mcpserver/tools_video_test.go` (RED):
  - Assert `video_generate` cost description and non-blocking job creation.
  - Assert `video_status` smart wait, directive output, and poster preview delivery.
  - Assert `video_cancel` release of budget reservation.
- [x] 3.3 Implement `internal/mcpserver/tools_video.go` and register tools in `internal/mcpserver/server.go` (GREEN).
- [x] 3.4 Register declarative resource `matriz://jobs/{id}` in `internal/mcpserver/resources.go`.

## PR-4: Manifest Scanner, Schema Export & Verification
- [x] 4.1 Update `internal/manifest/scan.go` to inventory `.mp4` and `.webm` assets and read video sidecars.
- [x] 4.2 Export MCP tool schemas to `~/.gemini/antigravity-cli/mcp/matriz/` (`video_generate.json`, `video_status.json`, `video_cancel.json`).
- [x] 4.3 Update `instructions.md` in MCP configuration with video tool usage patterns.
- [x] 4.4 Run full repository verification: `go test -v -race ./...` and `go build ./...`.
