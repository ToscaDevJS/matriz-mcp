# Core Pipeline Specification

## Purpose
Provides asset domain types, safe path resolution, deterministic image transformation, multi-format encoding, and sidecar metadata persistence.

## Requirements

### Requirement: Path Resolution and Traversal Defense
The system MUST resolve project-relative `AssetRef` handles to absolute paths and MUST reject any path that escapes the project root (including `..`, leading slashes, empty strings, and external symlinks).

#### Scenario: Valid project-relative path
- GIVEN a project root `/workspace`
- WHEN `ResolveRef("/workspace", "assets/hero.png")` is called
- THEN it returns `/workspace/assets/hero.png` and no error

#### Scenario: Traversal attempt rejected
- GIVEN a project root `/workspace`
- WHEN `ResolveRef("/workspace", "../etc/passwd")` or `ResolveRef("/workspace", "/etc/passwd")` is called
- THEN it returns an `ErrInvalidAssetRef` error stating `paths must stay inside the project root`

### Requirement: Deterministic Image Transformations
The system MUST perform image transformations (Crop, Resize, Adjust, Rotate, Sharpen) deterministically using pure Go / local algorithms without network calls.

#### Scenario: Image resizing
- GIVEN an image of 1000x500 pixels
- WHEN resized to a width of 400 pixels preserving aspect ratio
- THEN the resulting image is 400x200 pixels matching the golden fixture byte-for-byte

#### Scenario: Image adjustment
- GIVEN a valid source image
- WHEN brightness is adjusted by +20
- THEN the resulting image pixels match the expected golden fixture byte-for-byte

### Requirement: Multi-format Encoding
The system MUST encode images to JPEG, PNG, WebP, and AVIF formats without requiring external CGo compilation flags.

#### Scenario: WebP and AVIF encoding
- GIVEN an in-memory image
- WHEN encoded to WebP or AVIF with quality parameters
- THEN a valid byte stream for the target format is produced

### Requirement: Sidecar Metadata Management
The system MUST write and read a `.meta.json` sidecar alongside every produced asset conforming to the `matriz.sidecar/v1` schema.

#### Scenario: Round-trip sidecar persistence
- GIVEN a valid sidecar struct with schema `matriz.sidecar/v1`
- WHEN written to disk and reread
- THEN all fields (ref, origin, provider, model, prompt, seed, params, cost_usd) are preserved identically
