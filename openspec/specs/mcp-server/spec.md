# MCP Server Specification

## Purpose
Exposes image management tools and resources to LLMs over standard I/O MCP transport with thumbnails and actionable error handling.

## Requirements

### Requirement: Protocol Transport & Separation
The system MUST run exclusively over MCP `StdioTransport` without writing debug messages or unformatted text to `stdout`.

#### Scenario: Server startup
- GIVEN valid server configuration
- WHEN `matriz-mcp` is launched
- THEN it initializes tools and resources and communicates purely via JSON-RPC on stdio

### Requirement: Image Management Tools
The system MUST expose 5 tools: `img_list_models`, `img_transform`, `img_generate_drafts`, `img_refine`, and `img_upscale`, with deterministic tools marked `FREE` and generative tools marked `COSTS MONEY`.

#### Scenario: Generative tool description verification
- GIVEN the tool registry
- WHEN tool descriptions are inspected
- THEN generative tools start with `COSTS MONEY` and deterministic tools start with `FREE`

### Requirement: Dedicated Upscale Tool Exposure
The system MUST expose `img_upscale` to elevate low-resolution drafts into final production-grade assets using `CapabilityUpscale` and `ModelFinal`.

#### Scenario: Successful draft upscale
- GIVEN an existing draft asset
- WHEN `img_upscale` is invoked
- THEN the tool produces the upscaled asset on disk, writes a `.meta.json` sidecar with `derived_from`, and returns an `ImageContent` thumbnail <= 512px max edge

### Requirement: Thumbnail Return via Protocol
The system MUST return visual feedback as `*mcp.ImageContent` with dimensions capped at 512px max edge for every tool creating or modifying images, while keeping full-resolution files on disk.

#### Scenario: Tool thumbnail preview
- GIVEN a successful `img_transform` execution
- WHEN the tool result is returned
- THEN `Result.Content` contains an `ImageContent` with PNG MIME type and max edge <= 512px

### Requirement: Actionable Error Handling
The system MUST NOT propagate fatal Go errors to the MCP protocol; it MUST return `CallToolResult{IsError: true}` with structured actionable guidance.

#### Scenario: Missing asset error
- GIVEN a request to transform a non-existent asset `assets/missing.png`
- WHEN `img_transform` is invoked
- THEN it returns `IsError: true` with a message explaining what failed and suggesting checking `matriz://project/manifest`
