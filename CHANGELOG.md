# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **PR-0 (Scaffold & Core Types)**:
  - Core domain types: `Asset`, `AssetRef`, `Dimensions`, `Origin`, `Sidecar` (`internal/core/types.go`).
  - Actionable sentinel domain errors (`internal/core/errors.go`).
  - Path traversal defense: `ResolveRef` validating project boundaries (`internal/core/paths.go`).
  - Configuration loader purely from environment variables (`internal/config/config.go`).
  - Documentation of six design principles in Spanish (`docs/arquitectura.md`).
  - Strict TDD suite verifying dependency linking (`T-01`) and traversal defense (`T-02`).
