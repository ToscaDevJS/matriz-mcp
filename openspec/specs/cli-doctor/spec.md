# Spec: CLI Diagnostics (`cli-doctor`)

## Requirements

### Requirement: Runtime & Codec Diagnostics
The doctor command SHALL verify that all supported image codecs (PNG, JPEG, WebP, AVIF) encode and decode correctly in pure Go / WASM.

#### Scenario: Codec verification passes
- **Given** the runtime environment
- **When** `matriz doctor` runs codec checks
- **Then** it encodes and decodes a synthetic image for all 4 formats and reports `[✓] Runtime & Codecs`.

### Requirement: Configuration & Budget Diagnostics
The doctor command SHALL report provider status, configured models, session budget limits, and project root path.

#### Scenario: Configuration reporting
- **Given** runtime environment configuration
- **When** `matriz doctor` inspects configuration
- **Then** it validates `Provider`, `ModelDraft`, `ModelFinal`, `BudgetUSD`, and `ProjectRoot`.

### Requirement: Project Manifest & Asset Inventory Integrity
The doctor command SHALL inspect `matriz.json` (if present), validate `matriz.manifest/v1` schema, and verify that referenced assets exist on disk.

#### Scenario: Project validation in valid workspace
- **Given** a workspace with a valid `matriz.json` and existing assets
- **When** `matriz doctor` runs project integrity checks
- **Then** it confirms schema validity and asset counts without error.

### Requirement: MCP Client Integration Discovery
The doctor command SHALL search standard configuration paths for Claude Desktop and Cursor to verify if `matriz` or `matriz-mcp` is configured.

#### Scenario: MCP configuration inspection
- **Given** local user filesystem
- **When** `matriz doctor` runs MCP client discovery
- **Then** it reports whether client integrations were detected.
