# Design: CLI Diagnostics & Version Architecture

## Overview
This design introduces:
1. `internal/version`: Package containing build metadata (`Version`, `GitCommit`, `BuildDate`, `GoVersion`, `Platform`) and formatting utilities.
2. `internal/doctor`: Diagnostics package containing modular check probes (`ProbeCodecs`, `ProbeConfig`, `ProbeProject`, `ProbeMCPClients`), returning structured `CheckResult` instances (`Status`: Pass, Warn, Fail; `Message`, `Details`).
3. `cmd/matriz`: Unified CLI router parsing subcommands (`doctor`, `version`, `mcp`, `tui`).

## Component Architecture

```
cmd/matriz/
  main.go                     -> CLI router dispatching to version, doctor, mcp, or tui

internal/version/
  version.go                  -> Info struct, String(), JSON()

internal/doctor/
  doctor.go                   -> Doctor runner, Report aggregation, Terminal printer
  probes_codecs.go            -> WebP, AVIF, PNG, JPEG validation probe
  probes_config.go            -> Config, API key, provider probe
  probes_project.go           -> matriz.json and assets/ inventory probe
  probes_mcp.go               -> Claude Desktop, Cursor config detector
```

## Diagnostic Report Contract

```go
type Status string
const (
    StatusPass Status = "PASS"
    StatusWarn Status = "WARN"
    StatusFail Status = "FAIL"
)

type CheckResult struct {
    Name    string `json:"name"`
    Status  Status `json:"status"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

type Report struct {
    Version string        `json:"version"`
    Checks  []CheckResult `json:"checks"`
    Healthy bool          `json:"healthy"`
}
```

## Determinism & Zero Secrets
- API keys are never printed in full (only masked, e.g. `AIzaSy...XXXX`).
- Probes do not make paid generative model calls (cost is always $0.00).
