# Core Pipeline Specification (Delta)

## Purpose
Provides asset domain types, safe path resolution, deterministic image transformation, multi-format encoding, and sidecar metadata persistence.

## Requirements

### Requirement: Multi-format Encoding
The system MUST encode images to JPEG, PNG, and WebP formats without requiring external CGo compilation flags. The system MUST explicitly reject AVIF encoding requests with actionable guidance directing callers to WebP.

#### Scenario: WebP encoding
- GIVEN an in-memory image
- WHEN encoded to WebP with quality parameters
- THEN a valid byte stream for WebP is produced

#### Scenario: AVIF encoding rejection
- GIVEN a request to format or encode an image as AVIF
- WHEN `ParseFormat("avif")` or `ParseFormat(".avif")` is invoked
- THEN an error is returned stating that AVIF is disabled due to WASM encoding latency and recommending WebP
