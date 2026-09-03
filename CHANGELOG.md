# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed
- **`gen2brain/avif` dependency and AVIF encoding**: Pruned `github.com/gen2brain/avif` and `github.com/tetratelabs/wazero`. AVIF encoding is now explicitly rejected with actionable guidance directing callers to WebP (`.webp`), eliminating latency traps and reducing binary size.
- **`img_export_web` MCP tool** and its supporting pipeline
  (`internal/core/export.go`) — cut before the v0.1.0 release. AVIF encoding via
  `gen2brain/avif` (libaom on wazero) measured 62s for the 420/768/1024w set
  against 30ms for the same widths in WebP, which made responsive export
  unusable during draft iteration.
- **`srcset` string generation**, which had no consumer outside
  `img_export_web`. Single-width resize and format conversion remain available
  through `img_transform`, which infers the output format from the output file
  extension.

### Fixed
- **`img_generate_drafts` charged the budget for drafts it never delivered.**
  Gemini answered a `count: 4` request with a single image while the guard was
  debited for four. The settled cost is now derived from the images actually
  returned; the pre-flight reservation still prices the full request so the
  guard keeps failing closed (hard rule 7.11).
- **`count` never reached the Gemini API.** `CandidateCount` is now set from the
  requested draft count, so a four-draft request asks the model for four
  candidates instead of silently degrading to one.
- **Every sidecar in a multi-draft batch recorded the whole batch cost.** A
  `.meta.json` now records that image's share of the batch, not the total.
- **The requested shape never reached the Gemini API.** `ImageConfig` is now
  sent with the aspect ratio and resolution tier derived from the request, so a
  `16:9` draft is actually asked for as `16:9`. The tier is shared with the
  pricing table so the size requested and the size charged cannot drift apart.
- **Sidecars and returned assets recorded requested dimensions, not produced
  ones.** Gemini answered a 768x432 request with a 1408x768 image while every
  `.meta.json` claimed 768x432, and the model was handed the same wrong size to
  fit slots against. Both now describe the decoded file; when an image cannot be
  decoded the dimensions are omitted rather than back-filled from the request.
- **`CallTransform` and `CallGenerateDrafts` dropped the tool's structured
  output** whenever a result carried thumbnail content, leaving tests able to
  assert only on the preview.
- **`img_refine` wrote no sidecar at all.** Every refined image landed on disk
  with no record of the provider, model, prompt, seed or cost behind it, on the
  one operation that spends money (§5.4). A refined image now writes its
  `.meta.json` with `origin: generated` and `derived_from` pointing at the asset
  it started from.
- **`img_refine` could take the server down.** It indexed `result.Images[0]`
  without checking the length and discarded the `image.Decode` error before
  calling `Bounds()` on the result. Either one panicked, and a panic in a stdio
  handler kills the process instead of reaching the model; both now return an
  actionable tool error (hard rule 7.9).

### Added
- **PR-D3 (Unified CLI & SDD Complete)**:
  - Unified CLI binary `cmd/matriz/main.go` supporting `doctor`, `version`, `mcp`, `tui`, and `help`.
  - Full automated diagnostics for Go runtime, codecs, configuration, API keys, project structure, and MCP client discovery.
  - Complete SDD verification report with 100% test scenario pass rate.
- **PR-D2 (CLI Doctor Diagnostics Engine)**:
  - Multi-probe diagnostics engine in `internal/doctor/` inspecting pure-Go image codecs, configuration, masked API keys, project manifest, and MCP client discovery.
- **PR-D1 (CLI Version Package)**:
  - System and build metadata reporter `internal/version/` with plain text and JSON outputs.
- **PR-5 (Curation TUI & Preview)**:
  - Interactive Bubbletea TUI for navigating and reviewing asset catalogs (`internal/tui/model.go`, `cmd/matriz-tui/main.go`).
  - Key bindings and navigation controls (`internal/tui/keys.go`).
  - Local static HTML visual preview generator and browser launcher (`internal/preview/html.go`).
  - Strict TDD model test assertion `T-19` passing without filesystem side effects.
  - Verification report validated and admitted with 100% spec scenario coverage (`openspec/changes/matriz-v0-1-0/verify-report.md`).
- **PR-4 (Site Manifest & MCP Resource)**:
  - Manifest parser and validator for `matriz.json` matching `matriz.manifest/v1` schema (`internal/manifest/manifest.go`).
  - Filesystem asset scanner `ScanProject` combining disk images and `.meta.json` sidecars (`internal/manifest/scan.go`).
  - MCP Resource handler exposing `matriz://project/manifest` (`internal/mcpserver/resources.go`).
  - Manifest documentation in Spanish (`docs/manifiesto.md`).
  - Test assertions `T-16`, `T-17`, `T-18` passing.
- **PR-3 (MCP Server & Tools)**:
  - Local stdio MCP server entrypoint (`cmd/matriz-mcp/main.go`).
  - Four domain MCP tools: `img_list_models`, `img_transform`, `img_generate_drafts`, and `img_refine` (`internal/mcpserver/`).
  - Standardized thumbnail preview generation with 512px max edge returned as `*mcp.ImageContent` (`internal/mcpserver/thumbnail.go`).
  - Actionable error mapping returning `CallToolResult{IsError: true}` without breaking protocol stdio transport (`internal/mcpserver/errors.go`).
  - Tools reference and decision table documentation in Spanish (`docs/tools.md`).
  - Test assertions `T-11`, `T-12`, `T-13`, `T-14`, `T-15` passing.
- **PR-2 (Providers, Gemini & Budget Guard)**:
  - Generative `Provider` interface, capabilities, and request/result structures (`internal/providers/provider.go`).
  - Thread-safe `Guard` enforcing pre-flight budget and call ceilings failing closed (`internal/budget/budget.go`).
  - Deterministic offline `FakeProvider` for fast testing with zero network IO (`internal/providers/fake/fake.go`).
  - Google Gemini adapter over `google.golang.org/genai` for multimodal generation and editing (`internal/providers/gemini/gemini.go`).
  - Fixed-token pricing table with worst-case fail-closed fallback for unknown models (`internal/providers/gemini/pricing.go`).
  - Provider registry for dynamic resolution (`internal/providers/registry.go`).
  - Providers documentation with consultation date (`docs/proveedores.md`).
  - Tests `T-08`, `T-09`, `T-10`, `T-10b`, `T-10c`, `T-10d`.
- **PR-1 (Deterministic Pipeline)**:
  - Image transformation operations: `Crop`, `Resize`, `Adjust` (brightness/contrast/saturation), `Rotate`, `Sharpen` (`internal/core/transform.go`).
  - Multi-format encoding for JPEG, PNG, WebP, and AVIF via WASM/pure Go without CGo (`internal/core/encode.go`).
  - Responsive web export `ExportWeb` with configurable quality and `srcset` generator (`internal/core/export.go`).
  - Sidecar metadata reader and writer (`.meta.json`) conforming to `matriz.sidecar/v1` schema (`internal/core/sidecar.go`).
  - Byte-for-byte golden file test framework (`testdata/golden/`) passing tests `T-03`, `T-04`, `T-05`, `T-06`, `T-07`.
- **PR-0 (Scaffold & Core Types)**:
  - Core domain types: `Asset`, `AssetRef`, `Dimensions`, `Origin`, `Sidecar` (`internal/core/types.go`).
  - Actionable sentinel domain errors (`internal/core/errors.go`).
  - Path traversal defense: `ResolveRef` validating project boundaries (`internal/core/paths.go`).
  - Configuration loader purely from environment variables (`internal/config/config.go`).
  - Documentation of six design principles in Spanish (`docs/arquitectura.md`).
  - Strict TDD suite verifying dependency linking (`T-01`) and traversal defense (`T-02`).
