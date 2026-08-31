# Proposal: CLI Diagnostics & Version (`matriz doctor` & `matriz version`)

## Why
Developers and AI agents using Matriz need immediate diagnostic feedback when setting up or troubleshooting MCP connections, API keys, codecs, and project assets. Providing a dedicated `doctor` command and rich `version` metadata in a unified CLI eliminates configuration ambiguity and verifies ecosystem health.

## What
1. **Unified CLI (`cmd/matriz`)**:
   - Subcommands: `doctor`, `version`, `mcp`, `tui`.
   - Global flags: `--version` / `-v`, `--json`.
2. **`matriz version`**:
   - Prints SemVer, Git commit SHA, build timestamp, Go runtime, and Target OS/Arch.
   - Supports `--json` for machine readability.
3. **`matriz doctor`**:
   - Executes 5 diagnostic probes:
     1. **Runtime & Codecs**: Pure-Go / WASM encoder/decoder validation (PNG, JPEG, WebP, AVIF).
     2. **Configuration & Budget**: Active provider, model IDs, spend ceiling, call limits.
     3. **API Key & Network**: Checks `GOOGLE_API_KEY` format and availability.
     4. **Project Integrity**: Verifies `matriz.json` schema and assets inventory.
     5. **MCP Client Detection**: Inspects Claude Desktop / Cursor configuration files for Matriz registration.
   - Formatted output with pass/warn/fail indicators.

## Impact
- Non-breaking addition: `cmd/matriz-mcp` and `cmd/matriz-tui` remain functional.
- Zero extra external dependencies.
