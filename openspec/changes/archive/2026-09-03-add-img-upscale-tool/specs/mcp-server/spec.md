# MCP Server Specification (Delta: img_upscale)

## Purpose
Expose a dedicated generative upscaling and elevation tool (`img_upscale`) over MCP StdioTransport to elevate low-resolution drafts into final production-grade assets.

## Requirements

### Requirement: Dedicated Upscale Tool Exposure
The system MUST expose `img_upscale` alongside existing tools. The tool description MUST start with `COSTS MONEY`.

#### Scenario: Upscale tool registered
- GIVEN the MCP server tool definitions
- WHEN `GetToolDefinitions()` is inspected
- THEN `img_upscale` is present and its description starts with `COSTS MONEY`

### Requirement: Upscale Request Execution & Provenance
The system MUST accept a source draft `ref`, route to the final model (`cfg.ModelFinal`) using `CapabilityUpscale`, write the resulting image to disk, return a thumbnail (max edge <= 512px), and record sidecar metadata with `derived_from`.

#### Scenario: Successful draft upscale
- GIVEN an existing draft asset `assets/drafts/draft-1.png`
- WHEN `img_upscale` is invoked with `ref: "assets/drafts/draft-1.png"` and `output: "assets/finals/hero-pro.png"`
- THEN the tool produces `assets/finals/hero-pro.png`
- AND writes `assets/finals/hero-pro.png.meta.json` with `derived_from: "assets/drafts/draft-1.png"`
- AND returns a thumbnail `*mcp.ImageContent` with max edge <= 512px

### Requirement: Budget Enforcement
The system MUST enforce the budget ceiling before executing an upscale request and fail closed if insufficient funds remain.

#### Scenario: Budget exhausted on upscale
- GIVEN an exhausted budget guard ($0.00 remaining)
- WHEN `img_upscale` is invoked
- THEN it returns `IsError: true` without invoking the provider
