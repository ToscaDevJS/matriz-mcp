# Design: Elimination of AVIF Codec and Optimization of Deterministic Formats

## Context
Matriz was originally architected with four target codecs: JPEG, PNG, WebP, and AVIF. All were integrated using pure Go / WASM implementations to satisfy the Zero-CGo hard rule.

While WebP (`gen2brain/webp`) leverages a clean pure-Go/WASM bridge with negligible overhead (4–16ms for 1408px images), `gen2brain/avif` bundles `libaom` compiled to WASM on `wazero`. Without native vector SIMD instructions, encoding 1408x768 images requires 6.5s to 33.9s per file, totaling >62s sequentially.

Because MCP tools (`img_transform`) run synchronously across stdio JSON-RPC, long execution pauses stall client orchestrators (Claude Desktop, Cursor, Antigravity) and degrade user experience. Furthermore, `img_export_web` was previously removed for this exact reason (commit `f4dfb3e`).

## Architecture Decisions

### AD-1: Explicit Rejection with Actionable Error Message in `ParseFormat`
Instead of silently failing or dropping AVIF without guidance, `core.ParseFormat("avif")` and `.avif` extensions will explicitly return an actionable error:
```
unsupported image format "avif": AVIF encoding is disabled due to WASM libaom latency (30s+); use "webp", "png", or "jpeg"
```
This directly informs calling AI agents why their request cannot be fulfilled in AVIF and instructs them to switch to WebP.

### AD-2: Prune `gen2brain/avif` and `tetratelabs/wazero`
Dropping `gen2brain/avif` from `go.mod` removes the dependency tree including `wazero`. Note: verify whether `gen2brain/webp` also depends on `wazero` or if it uses its own runtime; `go mod tidy` will retain only the minimal necessary set.

### AD-3: Align `matriz doctor` Probes
`doctor.ProbeCodecs` will encode test fixtures in memory across the 3 supported formats: PNG, JPEG, and WebP. The diagnostic output will report:
`runtime operational; PNG, JPEG, WebP codecs verified`.

## Trade-offs
- **Pros**: Zero MCP timeouts, faster compilation, smaller binary, no latency traps, 100% reliable deterministic tools.
- **Cons**: Matriz will not directly output `.avif` files. However, WebP achieves ~97%+ browser compatibility and ~30% smaller footprint than JPEG, making it the premier web format for autonomous agents today.
