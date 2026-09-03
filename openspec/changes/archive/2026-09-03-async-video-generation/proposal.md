# Proposal: Asynchronous Video Generation via Gemini Omni Flash and Veo 3.1

## Why
Matriz currently provides high-fidelity curation and generation for static raster assets (drafts, refinement, upscaling, and local image transforms). However, modern web landing pages, heroes, and rich presentation sections increasingly demand short animated visuals, looping video backgrounds, and motion assets directly derived from brand imagery.

Video generation models (such as Gemini Omni Flash for fast multimodal iteration and Veo 3.1 for cinematic production quality) have operational latencies between 30 seconds and several minutes. Because LLM agent interactions and MCP tool calls cannot block synchronously without risking client timeouts or freezing agent execution, Matriz requires a first-class asynchronous job orchestration pattern, dedicated video provider abstractions, and explicit budget safety controls.

## What
1. **Asynchronous Job Management (`internal/jobs`)**:
   - Introduce an in-memory / persisted job tracker with state lifecycle (`queued` -> `running` -> `completed` -> `failed`).
   - Allow non-blocking initiation and explicit status polling.

2. **Video Provider Abstraction (`internal/providers`)**:
   - Introduce `VideoProvider` interface separated from static `Provider` to maintain Single Responsibility Principle.
   - Support `CapabilityVideoDraft` (Gemini Omni Flash), `CapabilityVideoFinal` (Veo 3.1), and `CapabilityImageToVideo`.
   - Implement Long Running Operation (LRO) polling adapters for Google GenAI / Vertex endpoints.

3. **MCP Tool Suite (`internal/mcpserver`)**:
   - `vid_generate`: Initiates an asynchronous video generation task (prompt, source image slot, duration, model tier) and returns immediately with a `job_id`.
   - `vid_check_job`: Non-blocking status lookup returning current state, elapsed time, and result URI when ready.
   - Resource exposure `matriz://jobs/{id}` for declarative job inspection.

4. **Budget & Safety Guard (`internal/budget`)**:
   - Extend `budget.Guard` to support two-phase reservations: reserve estimated cost on submission, settle or refund on final state.

5. **Storage & Sidecar Metadata**:
   - Output videos as MP4/WebM under the project assets tree with structured `.meta.json` sidecars preserving generation prompt, seed, source image ref, and model tier.

## Capabilities Contract
- New capabilities:
  - `CapabilityVideoDraft` ("video_draft"): Fast low-resolution drafts (Gemini Omni Flash).
  - `CapabilityVideoFinal` ("video_final"): High-fidelity cinematic renders (Veo 3.1).
  - `CapabilityImageToVideo` ("image_to_video"): Direct animation from an existing static image asset.
- Provider implementations: Google Gemini/Veo provider, Fake provider for deterministic testing.

## Rollback Plan
- Video tools (`vid_generate`, `vid_check_job`) are additive to the MCP tool registry. If rolled back, unregister the tools and provider instances; existing static image pipelines and manifests remain unaffected.
