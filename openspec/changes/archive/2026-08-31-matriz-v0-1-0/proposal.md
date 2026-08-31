# Proposal: Matriz Core & MCP Server (v0.1.0)

## Intent

Build **Matriz**, a local MCP stdio server and companion TUI in Go for managing client website image assets. Matriz normalizes heterogeneous customer photos into responsive, optimized web formats and provides controlled generative AI enhancements (Gemini Nano Banana) with strict budget guards.

## Scope

### In Scope
- Core domain model, path traversal guards, and project resolution (`internal/core`).
- Deterministic image processing pipeline: crop, resize, adjust, encode (JPEG/PNG/WebP/AVIF), and responsive export (`internal/core`).
- Generative provider abstraction with Google Gemini (`google.golang.org/genai`) and deterministic `fakeProvider` for offline testing (`internal/providers`).
- Pre-flight budget guard enforcing spend and call limits (`internal/budget`).
- MCP stdio server exposing 5 tools (`img_list_models`, `img_transform`, `img_export_web`, `img_generate_drafts`, `img_refine`) returning 512px thumbnails and structured metadata (`internal/mcpserver`, `cmd/matriz-mcp`).
- Site manifest resource (`matriz://project/manifest`) and filesystem inventory scanner (`internal/manifest`).
- Interactive TUI for asset review and local HTML preview (`cmd/matriz-tui`, `internal/tui`, `internal/preview`).
- Comprehensive table-driven tests, golden file fixtures, and strict TDD discipline.

### Out of Scope
- Remote HTTP transport, authentication, and multi-tenancy.
- In-place modification of client source assets.
- General-purpose image manipulation beyond web delivery requirements.
- PR-6: fal.ai integration (deferred to future milestone per explicit triggers).

## Capabilities

### New Capabilities
- `core-pipeline`: Project asset abstractions, path security, deterministic transforms, responsive export, and sidecar metadata.
- `provider-budget`: Generative provider interface, Gemini adapter, fake test provider, and budget guard.
- `mcp-server`: MCP stdio transport, 5 domain tools with thumbnail previews, and error mapping.
- `site-manifest`: Manifest schema (`matriz.json`), MCP resource provider, and disk scanner.
- `curation-tui`: Terminal UI model and local HTML preview for asset inspection.

### Modified Capabilities
None

## Approach

Follow Clean Architecture ("one core, two frontends"). Implement 6 chained PR units (PR-0 to PR-5) following strict TDD (RED -> GREEN -> REFACTOR). Pure Go / WASM image codecs with zero CGo. Pre-flight cost estimation to fail closed on budgets.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `go.mod`, `go.sum` | New | Module initialization with MCP and imaging dependencies |
| `internal/core/` | New | Types, errors, paths, transforms, encoders, export, sidecars |
| `internal/providers/` | New | Provider interfaces, registry, fake, and Gemini implementation |
| `internal/budget/` | New | Spending limit guard |
| `internal/mcpserver/`, `cmd/matriz-mcp/` | New | MCP server, tools, thumbnails, resources |
| `internal/manifest/` | New | Manifest parsing, validation, and disk scanner |
| `internal/tui/`, `internal/preview/`, `cmd/matriz-tui/` | New | Bubbletea TUI and HTML preview |
| `docs/` | New | Architecture, tools, manifest, and providers documentation |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Codec compatibility on target arch | Low | Spike in PR-0 with pure Go / WASM fallbacks |
| Cost overrun from autonomous agents | High | Pre-flight `Guard.Reserve` before all generative calls |
| Path traversal via LLM inputs | Medium | Mandatory `ResolveRef` validation for all asset references |
| Gemini visible watermark in free tier | Medium | Verification step A-03 documented and flagged in docs |

## Rollback Plan

Revert Git commits / branches per PR unit. Project is greenfield with zero external dependencies to roll back.

## Dependencies

- Go 1.25+ (verified Go 1.26.3)
- `github.com/modelcontextprotocol/go-sdk` v1.7.0+
- `google.golang.org/genai`

## Success Criteria

- [ ] All 6 PR units compile cleanly (`go build ./...`, `go vet ./...`).
- [ ] 100% pass on 23 test assertions (`T-01` through `T-19`) with zero test failures.
- [ ] Manual acceptance criteria `A-01` (MCP Inspector thumbnail render) verified.
- [ ] Zero API keys or secrets in repository or logs.
