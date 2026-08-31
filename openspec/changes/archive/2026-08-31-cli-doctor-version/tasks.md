# Tasks: CLI Diagnostics & Version (`cli-doctor-version`)

## Phase 1: Version Package & Tests (PR-D1)
- [x] 1.1 RED: Write `internal/version/version_test.go` checking plain and JSON output formatting
- [x] 1.2 GREEN: Implement `internal/version/version.go`
- [x] 1.3 REFACTOR: Verify build info injection compatibility with `ldflags`

## Phase 2: Doctor Probes & Engine (PR-D2)
- [x] 2.1 RED: Write `internal/doctor/doctor_test.go` covering codec probe, config probe, and project integrity probe
- [x] 2.2 GREEN: Implement `internal/doctor/` (`doctor.go`, `probes_codecs.go`, `probes_config.go`, `probes_project.go`, `probes_mcp.go`)
- [x] 2.3 REFACTOR: Verify masking of API keys and zero network cost

## Phase 3: Unified CLI Entrypoint & Verification (PR-D3)
- [x] 3.1 RED: Write integration test for `cmd/matriz` subcommands in `cmd/matriz/main_test.go`
- [x] 3.2 GREEN: Implement `cmd/matriz/main.go` with subcommand routing (`doctor`, `version`, `mcp`, `tui`)
- [x] 3.3 REFACTOR: Run full test suite `go test ./...`, `go vet ./...`, verify SDD report and archive
