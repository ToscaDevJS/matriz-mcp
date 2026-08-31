# Spec: CLI Version (`cli-version`)

## Requirements

### Requirement: Version Metadata Reporting
The system SHALL expose version information including semantic version, git commit, build date, Go version, and platform architecture.

#### Scenario: Plain Text Version Output
- **Given** the binary is executed with `matriz version` or `matriz --version`
- **When** version information is requested
- **Then** stdout displays `matriz v0.1.0` and compilation details.

#### Scenario: JSON Formatted Version Output
- **Given** the binary is executed with `matriz version --json`
- **When** json format is specified
- **Then** stdout outputs a valid JSON object with `version`, `commit`, `build_date`, `go_version`, and `platform`.
