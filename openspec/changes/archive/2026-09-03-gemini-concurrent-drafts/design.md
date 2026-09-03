# Design: Concurrent Fan-Out Architecture for Gemini Image Generation

## Context
Google Gemini's multimodal generation API handles text and code generation with `CandidateCount`. However, the Imagen multimodal endpoint treats image generation as a discrete 1-candidate pipeline per inference invocation. Setting `CandidateCount: 4` in `genai.GenerateContentConfig` yields exactly 1 image in `resp.Candidates[0].Content.Parts`.

Because an essential value proposition of `img_generate_drafts` is to provide quick visual choices (1 to 4 drafts) in a single tool call, the adapter layer must bridge this capability gap.

## Architectural Decisions

### AD-1: Goroutine Fan-out with Concurrency Bounding
When `req.Count > 1`:
- Spawn worker goroutines to perform individual single-image generation requests (`Count: 1`).
- Protect against rate-limiting by bounding maximum concurrent requests to `min(req.Count, 4)`.
- Use `context.WithCancel` to abort pending requests if context expires or is cancelled.

### AD-2: Seed Offsetting for Reproducible Diversity
If the caller provides an explicit `req.Seed`:
- Worker $i \in [0, N-1]$ receives `seed: *req.Seed + int64(i)`.
- If `req.Seed` is `nil`, each worker request lets the provider choose a random seed independently.
- This prevents the model from generating $N$ identical images when a user specifies a seed.

### AD-3: Best-Effort Resilience (Tolerating Partial Failures)
When generating 4 images:
- Collect results into a thread-safe slice protected by a mutex.
- If at least 1 image succeeds, the overall operation succeeds and returns the collected images.
- Settle cost strictly for `len(deliveredImages)`:
  `settledCostUSD(req, len(deliveredImages))`.
- If 0 images succeed, return the first encountered error.

## Alternatives Considered
- *Sequential execution in a loop*: Takes $4 \times \approx 3\text{s} = 12\text{s}$, creating noticeable latency for the user. Parallel execution completes in $\approx 3\text{s}$ total.
- *Failing completely on any worker error*: If 3 succeed and 1 experiences a transient network hiccup, discarding the 3 good images wastes network spend and degrades UX. Best-effort is strictly superior.
