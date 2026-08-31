# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **PR-2 (Providers, Gemini & Budget Guard)**:
  - Generative `Provider` interface, capabilities, and request/result structures (`internal/providers/provider.go`).
  - Thread-safe `Guard` enforcing pre-flight budget and call ceilings failing closed (`internal/budget/budget.go`).
  - Deterministic offline `FakeProvider` for fast testing with zero network IO (`internal/providers/fake/fake.go`).
  - Google Gemini adapter over `google.golang.org/genai` for multimodal generation and editing (`internal/providers/gemini/gemini.go`).
  - Fixed-token pricing table with worst-case fail-closed fallback for unknown models (`internal/providers/gemini/pricing.go`).
  - Provider registry for dynamic resolution (`internal/providers/registry.go`).
  - Providers documentation with consultation date (`docs/proveedores.md`).
  - Tests `T-08`, `T-09`, `T-10`, `T-10b`, `T-10c`, `T-10d`.
- **PR-1 (Deterministic Pipeline)**:
  - Image transformation operations: `Crop`, `Resize`, `Adjust` (brightness/contrast/saturation), `Rotate`, `Sharpen` (`internal/core/transform.go`).
  - Multi-format encoding for JPEG, PNG, WebP, and AVIF via WASM/pure Go without CGo (`internal/core/encode.go`).
  - Responsive web export `ExportWeb` with configurable quality and `srcset` generator (`internal/core/export.go`).
  - Sidecar metadata reader and writer (`.meta.json`) conforming to `matriz.sidecar/v1` schema (`internal/core/sidecar.go`).
  - Byte-for-byte golden file test framework (`testdata/golden/`) passing tests `T-03`, `T-04`, `T-05`, `T-06`, `T-07`.
- **PR-0 (Scaffold & Core Types)**:
  - Core domain types: `Asset`, `AssetRef`, `Dimensions`, `Origin`, `Sidecar` (`internal/core/types.go`).
  - Actionable sentinel domain errors (`internal/core/errors.go`).
  - Path traversal defense: `ResolveRef` validating project boundaries (`internal/core/paths.go`).
  - Configuration loader purely from environment variables (`internal/config/config.go`).
  - Documentation of six design principles in Spanish (`docs/arquitectura.md`).
  - Strict TDD suite verifying dependency linking (`T-01`) and traversal defense (`T-02`).
