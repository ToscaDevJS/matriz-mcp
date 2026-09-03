# Proposal: Concurrent Multi-Draft Generation for Gemini Provider

## Why
In live API testing against Google AI Studio with `gemini-3.1-flash-lite-image`, the Gemini image generation endpoint (`Models.GenerateContent`) only returns 1 candidate image per API call, ignoring `GenerateContentConfig.CandidateCount > 1`.

When an MCP client or AI agent invokes `img_generate_drafts` requesting $N$ drafts (e.g. `count: 4`), expecting simultaneous stylistic variations to choose from, only 1 image was delivered.

To fulfill multi-draft requests reliably without depending on unverified upstream candidate multiplexing, the Gemini provider must orchestrate concurrent parallel requests in Go using goroutines.

## What
1. **Parallel Fan-out in `internal/providers/gemini/gemini.go`**:
   - When `req.Count > 1`, launch $N$ concurrent calls in parallel using bounded worker goroutines (`sync.WaitGroup` or `errgroup.Group`).
   - For each concurrent request, assign individual seed variations when a base seed is supplied (`base_seed + i`) to guarantee divergent outputs.
2. **Resilience & Partial Failure Handling**:
   - If one of the parallel requests fails (e.g. transient 429 / network glitch) while others succeed, return the successfully delivered images instead of failing the entire batch.
   - If all concurrent requests fail, return a descriptive error.
3. **Settled Cost Proportionality**:
   - Settle budget charges strictly according to the actual number of delivered images: `settledCostUSD(req, len(images))`.
4. **Offline Test Suite (TDD)**:
   - Expand table-driven tests in `internal/providers/gemini/gemini_test.go` and `internal/providers/fake/fake.go` to assert concurrent fan-out, partial failure recovery, and seed distribution.

## Rollback Plan
If upstream Gemini adds multi-candidate support to image models, a feature flag or single-call multiplexer can be re-enabled without changing the `providers.Provider` interface contract.
