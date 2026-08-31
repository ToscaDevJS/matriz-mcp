# Design: Matriz Core & MCP Architecture

## Technical Approach

Matriz follows Hexagonal / Clean Architecture: business logic, transformations, and metadata sidecars live exclusively in `internal/core`. Two decoupled frontends (`cmd/matriz-mcp` and `cmd/matriz-tui`) consume this core through structured interfaces. All image codecs are pure Go or WASM-based to avoid CGo dependencies.

## Architecture Decisions

| Decision | Option Chosen | Tradeoff / Alternatives Rejected | Rationale |
|---|---|---|---|
| Image Engine Architecture | Single core (`internal/core`) with 2 frontends | Separate MCP and TUI business logic | Prevents divergence between TUI curation and MCP agent behavior (P3). |
| Codec Strategy | Pure Go / WASM (`imaging`, `gen2brain/webp`, `gen2brain/avif`) | CGo-based `libvips` / `libwebp` | Zero runtime CGo dependencies ensures cross-platform portability. |
| Generative AI Provider | Google Gemini via official Go SDK | fal.ai / OpenAI | Official Go SDK available, token-based fixed pricing, and free development tier. |
| Budget Enforcement | Pre-flight `Guard.Reserve` + worst-case fallback | Post-call billing alerts | Prevents unbounded agent spending loops before making network requests. |
| TUI Image Preview | Local browser HTML preview on Enter | Sixels / Kitty / iTerm graphics | Universally compatible across all terminals without native protocol flakes. |

## Data Flow

```
LLM Agent / User ──→ cmd/matriz-mcp (stdio) ──→ internal/mcpserver
                           │
                           ├──→ ResolveRef (Path Guard)
                           ├──→ internal/core (Transforms, Encoders, Sidecars)
                           ├──→ internal/budget (Pre-flight check)
                           └──→ internal/providers (Gemini / Fake)
```

## Interfaces / Contracts

### 1. `Provider` Interface (`internal/providers/provider.go`)
```go
type Provider interface {
	Name() string
	Capabilities() []Capability
	EstimateCostUSD(req GenerateRequest) float64
	Generate(ctx context.Context, req GenerateRequest) (*Result, error)
	Edit(ctx context.Context, req EditRequest) (*Result, error)
}
```

### 2. Core Types (`internal/core/types.go`)
```go
type Origin string
const (
	OriginClient    Origin = "client"
	OriginGenerated Origin = "generated"
	OriginDerived   Origin = "derived"
)

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type AssetRef string

type Asset struct {
	Ref      AssetRef   `json:"ref"`
	Origin   Origin     `json:"origin"`
	MIMEType string     `json:"mime_type"`
	Dims     Dimensions `json:"dims"`
	Bytes    int64      `json:"bytes"`
}
```

## Threat Matrix

| Threat | Applicable | Safe / Mitigation Behavior | Planned RED Test |
|---|---|---|---|
| Path Traversal (`../etc/passwd`, `/etc/passwd`) | Yes | `ResolveRef` strips/rejects traversing refs and symlinks escaping root | `T-02`: `TestResolveRef_RejectsTraversal` |
| Budget Exhaustion Loop | Yes | `Guard.Reserve` fails closed on limits or max calls | `T-09`, `T-10`: `TestGuard_RejectsWhenExceeded` |
| Unknown Model Cost Leak | Yes | `EstimateCostUSD` falls back to worst-case known price, never $0 | `T-10b`: `TestEstimateCostUSD_UnknownModelFallback` |
| Stdio Pollution | Yes | No `fmt.Print` to stdout in server; only stderr for logs | `T-15`: Grep assertion on stdio usage |

## Testing Strategy

- **Strict TDD (RED -> GREEN -> REFACTOR)**: Write failing test, verify failure, implement minimal code, verify green, refactor.
- **Golden Files (`testdata/golden/`)**: Byte-for-byte image comparison for deterministic transforms (`Resize`, `Adjust`).
- **Offline Fake Provider (`fakeProvider`)**: Deterministic hashes, invocation tracking, zero network IO.
