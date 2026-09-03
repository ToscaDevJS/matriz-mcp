# Proposal: Dedicated img_upscale Tool for Draft-to-Pro Elevation

## Why
In the two-stage image generation architecture defined in Matriz (§5.5), drafts are generated at low resolution with the lightweight model (`gemini-3.1-flash-lite-image`) for fast and inexpensive concept exploration. Once a draft is selected, it must be elevated to high-resolution production quality using the final model (`gemini-3-pro-image-preview`).

Previously, this elevation could only be accomplished by hijacking `img_refine` with `operation: "inpaint"` without a mask, or by relying on extensive source inspection. This overloaded `img_refine` (which is intended for localized inpainting, outpainting, and background removal) and created semantic friction and ambiguity for LLM agents.

Exposing an explicit `img_upscale` tool aligns directly with the Single Responsibility Principle, matches `CapabilityUpscale` already defined in `internal/providers`, and provides an unambiguous interface for AI agents.

## What
1. **MCP Tool Registration (`internal/mcpserver/server.go`)**:
   - Register `ToolUpscale` (`img_upscale`) with description starting with `COSTS MONEY`.
   - Add handler routing in `RegisterTools`.
   - Expose test helper `CallUpscale`.
2. **Tool Handler Implementation (`internal/mcpserver/tools_generative.go`)**:
   - Implement `handleUpscale`:
     - Validate input draft `ref` via `core.ResolveRef`.
     - Request budget reservation against `cfg.ModelFinal`.
     - Dispatch `provider.Edit` with `providers.CapabilityUpscale`.
     - Write upscaled image to project-relative `output` path.
     - Generate visual thumbnail preview (max edge 512px).
     - Write provenance sidecar (`.meta.json`) recording `derived_from` source draft.
3. **MCP Tool Schema Exposure**:
   - Create schema definition `~/.gemini/antigravity-cli/mcp/matriz/img_upscale.json`.
   - Update `instructions.md` to reference `img_upscale`.
4. **Test Suite Expansion (TDD)**:
   - Add unit tests in `internal/mcpserver/tools_generative_test.go` and `internal/mcpserver/server_test.go` verifying tool description cost marker, thumbnail output, sidecar provenance, and error handling.

## Capabilities Contract
- Generative capability utilized: `providers.CapabilityUpscale` ("upscale").
- Provider compatibility: Providers reporting `CapabilityUpscale` (e.g. Gemini, Fake).
- Model routing: Always dispatches using `cfg.ModelFinal`.

## Rollback Plan
If `img_upscale` needs to be rolled back, remove `ToolUpscale` from `RegisterTools` and `GetToolDefinitions()`. Existing drafts and refined assets remain untouched.
