# Site Manifest Specification

## Purpose
Provides structured context on website image slots, design palette, and project asset inventory via a dedicated MCP resource.

## Requirements

### Requirement: Manifest Validation
The system MUST validate `matriz.json` files against the `matriz.manifest/v1` schema, requiring explicit `origin` for every asset.

#### Scenario: Missing origin validation failure
- GIVEN a manifest missing the `origin` field on an asset entry
- WHEN the manifest is parsed and validated
- THEN validation fails with an actionable error message

### Requirement: Filesystem Inventory Reconstruction
The system MUST scan the project assets directory and reconstruct the asset inventory by reading disk files and sidecar metadata.

#### Scenario: Full disk scan
- GIVEN a directory containing generated and transformed assets with `.meta.json` sidecars
- WHEN `ScanAssets` is run
- THEN all assets and their dimensions, origins, and MIME types are indexed accurately

### Requirement: MCP Resource Exposure
The system MUST expose `matriz://project/manifest` as a readable JSON MCP resource.

#### Scenario: Read manifest resource
- GIVEN a running MCP server with an initialized project
- WHEN an LLM requests resource `matriz://project/manifest`
- THEN it receives the complete JSON manifest conforming to `matriz.manifest/v1`
