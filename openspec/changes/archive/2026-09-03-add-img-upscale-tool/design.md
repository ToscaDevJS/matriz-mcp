# Design: Dedicated img_upscale Tool

## Context
Matriz implements a two-stage generative image pipeline:
1. Low-resolution drafts via `img_generate_drafts` using a lightweight, fast, inexpensive model (`gemini-3.1-flash-lite-image`).
2. High-resolution final assets elevated via the final flagship model (`gemini-3-pro-image-preview`).

While `CapabilityUpscale` was defined in `internal/providers`, no corresponding dedicated MCP tool existed. Users and LLM agents had to synthesize an inpaint operation via `img_refine` to upscale.

## Architectural Decisions

### AD-1: Dedicated MCP Tool vs Overloaded img_refine
- **Decision**: Create `img_upscale` as an explicit, first-class MCP tool.
- **Rationale**: Follows the Single Responsibility Principle. Separates localized composition editing (`img_refine`: inpaint, outpaint, remove background) from full-image fidelity elevation (`img_upscale`). Eliminates model ambiguity and reduces prompt manipulation.

### AD-2: Input and Output Contract
- **Input (`UpscaleIn`)**:
  - `ref` (string, required): project-relative path of the draft asset.
  - `prompt` (string, optional): specific enhancement guidance (defaults to a high-fidelity elevation prompt if omitted).
  - `output` (string, required): project-relative destination path.
  - `seed` (*int64, optional): optional seed for reproducibility.
- **Output (`UpscaleOut`)**:
  - `asset` (`core.Asset`): asset metadata (ref, mime, dimensions, bytes).
  - `cost_usd` (float64): actual spent cost.
  - `budget_left_usd` (float64): remaining session budget.

### AD-3: Lifecycle & Provenance Sidecars
- Follows rule §5.4: every produced file writes a `<output>.meta.json` sidecar.
- The sidecar records:
  - `origin`: `"generated"`
  - `derived_from`: the source draft `ref`
  - `provider`: active provider name
  - `model`: `cfg.ModelFinal`
  - `cost_usd`: cost charged
  - `params`: dimensions, operation `"upscale"`, seed

### AD-4: Thumbnail Protocol Compliance
- In accordance with rule §5.12, the full-size image is saved directly to disk.
- Only a thumbnail with `maxEdge <= 512px` is returned over the MCP stdio wire as `*mcp.ImageContent`.

## Alternatives Considered
- *Overloading `img_refine` with an `upscale` operation*: Kept the tool count at 4, but confused agents since `img_refine` requires specifying masks or inpainting prompts, leading to unnecessary filesystem exploration and errors.
