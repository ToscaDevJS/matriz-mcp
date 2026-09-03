# Proposal: Remove AVIF Codec and Streamline Web Codecs

## Why
`gen2brain/avif` compiles `libaom` to WebAssembly on top of `wazero` without native SIMD acceleration. Real-world benchmarks on 1408x768 images demonstrate sequential encoding times exceeding 62 seconds (vs 30ms for the equivalent WebP set) — an approximately 2000x latency penalty.

When invoked via the stdio MCP protocol (`img_transform`), this latency blocks the JSON-RPC communication loop for 30+ seconds, creating severe risk of client timeouts and stalled agent workflows.

Removing `gen2brain/avif`:
1. Eliminates the severe latency trap across all MCP tools.
2. Removes heavy WASM runtime transitive dependencies (`github.com/tetratelabs/wazero`).
3. Reduces compiled binary size and build time.
4. Focuses Matriz's deterministic core on blazing-fast (<20ms), battle-tested web formats: **WebP**, **PNG**, and **JPEG**.

## What
1. **Dependency Cleanup**:
   - Drop `github.com/gen2brain/avif` from `go.mod` and run `go mod tidy` to prune `tetratelabs/wazero`.
2. **Core Pipeline Update (`internal/core`)**:
   - Remove `FormatAVIF` as an active encoding target in `Encode`.
   - Update `ParseFormat`: Return an actionable error explaining that AVIF is disabled due to WASM encoding latency and recommending WebP.
   - Update `internal/core/encode_test.go` to assert supported formats (PNG, JPEG, WebP) and verify AVIF rejection.
3. **Doctor Diagnostics Update (`internal/doctor`)**:
   - Update `ProbeCodecs` to test the verified supported suite: PNG, JPEG, and WebP.
   - Update doctor test assertions in `internal/doctor/doctor_test.go`.
4. **Documentation & Spec Synchronization**:
   - Update `openspec/specs/core-pipeline/spec.md`.
   - Update `README.md` and resolve item #4 in `PENDING.md`.
   - Update `CHANGELOG.md`.

## Rollback Plan
If native pure-Go SIMD AVIF encoding becomes viable in the future (or if an external optional sidecar utility is adopted), AVIF encoding can be reintroduced behind an explicit opt-in capability flag without breaking the core pipeline interface.
