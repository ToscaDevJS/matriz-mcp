# Video Generation Specification (Delta: Async Video Generation)

## Purpose
Expose asynchronous video generation capabilities (Text-to-Video and Image-to-Video) over MCP StdioTransport and the Go core, supporting rapid drafting via Gemini Omni Flash and cinematic high-fidelity rendering via Veo 3.1 without blocking client execution.

## Requirements

### Requirement: Asynchronous Video Generation Tool Exposure
The system MUST expose `video_generate` alongside existing image tools. The tool description MUST start with `COSTS MONEY and takes minutes`. The tool MUST return immediately with a unique `job_id`, status `processing`, and estimated turnaround time in seconds.

#### Scenario: Video generation tool registered
- GIVEN the MCP server tool definitions
- WHEN `GetToolDefinitions()` is inspected
- THEN `video_generate` is present and its description starts with `COSTS MONEY and takes minutes`

#### Scenario: Non-blocking initiation of video generation
- GIVEN an initialized MCP server with available budget
- WHEN `video_generate` is invoked with a valid prompt, duration, and model tier
- THEN the tool returns within 1 second with `status: "processing"`
- AND provides a valid `job_id` and `poll_interval_sec`

---

### Requirement: Two-Phase Budget Guard Reservation & Settlement
The system MUST reserve estimated cost prior to dispatching any generation request and generate a reservation ticket. If the budget is exhausted, the request MUST fail closed. When a job completes, the system MUST commit actual spend against the ticket. If a job fails or is cancelled, the system MUST release the hold in full.

#### Scenario: Budget reservation on job dispatch
- GIVEN a budget limit of $5.00 and spend of $0.00
- WHEN `video_generate` is called with an estimated cost of $1.50
- THEN the budget guard records a reservation hold of $1.50
- AND `BudgetLeft()` reports $3.50 available

#### Scenario: Budget release on failed job
- GIVEN an active job holding a $1.50 reservation
- WHEN the job status transitions to `failed`
- THEN the reservation hold is released
- AND `BudgetLeft()` returns to the pre-reservation capacity without recording spend

#### Scenario: Budget commitment on successful job
- GIVEN an active job holding a $1.50 reservation
- WHEN the job status transitions to `completed` with actual cost $1.40
- THEN the reservation hold is removed
- AND actual spend increases by $1.40
- AND completed call count increments by 1

---

### Requirement: Image-to-Video Animation & Provenance Tracking
The system MUST support animating an existing project image asset (`CapabilityImageToVideo`). The resulting video sidecar MUST record `derived_from` referencing the original asset's relative path.

#### Scenario: Image-to-Video generation and sidecar recording
- GIVEN an existing image asset `assets/hero.png`
- WHEN `video_generate` is invoked with `ref: "assets/hero.png"` and prompt `"animate gentle wind and subtle camera push"`
- THEN the provider receives the image bytes and motion prompt
- AND upon completion produces an MP4 video under `assets/videos/`
- AND writes a sidecar `.meta.json` where `derived_from` equals `"assets/hero.png"`

---

### Requirement: Text-to-Video Generation
The system MUST support direct video synthesis from text prompts (`CapabilityTextToVideo`) without requiring an anchor image.

#### Scenario: Text-to-Video generation
- GIVEN a text prompt `"Cinematic slow-motion shot of espresso dripping into glass cup"`
- WHEN `video_generate` is invoked without a source `ref`
- THEN the provider receives the prompt, duration, and aspect ratio
- AND generates an MP4 video under `assets/videos/` with `origin: "generated"` and empty `derived_from`

---

### Requirement: Job Polling with Bounded Smart Wait
The system MUST expose `video_status` marked as `FREE`. The tool MUST accept an optional `wait_seconds` parameter (default 5, clamped between 0 and 10 seconds). If the job is still processing, the tool MUST hold response execution up to `wait_seconds` to pace agent turn loops. When the job completes, it MUST return asset metadata and an `ImageContent` preview thumbnail (max edge <= 512px).

#### Scenario: Polling an in-progress job with smart wait
- GIVEN an in-progress job `job-123`
- WHEN `video_status` is invoked with `job_id: "job-123"` and `wait_seconds: 5`
- THEN the server holds execution for up to 5 seconds before returning
- AND returns `status: "processing"` with an actionable directive advising the agent not to poll in a tight loop

#### Scenario: Successful job completion status
- GIVEN a completed job `job-123`
- WHEN `video_status` is invoked
- THEN it returns `status: "completed"`
- AND returns the created `core.Asset` with relative path, duration, and dimensions
- AND includes an `*mcp.ImageContent` thumbnail poster (max edge <= 512px)

---

### Requirement: Job Cancellation
The system MUST expose `video_cancel` marked as `FREE`. Cancelling a processing job MUST notify the provider, release any budget hold, and mark the job as `cancelled`.

#### Scenario: Cancelling an in-flight job
- GIVEN an active in-flight job with a budget hold
- WHEN `video_cancel` is invoked with the `job_id`
- THEN the job status is set to `cancelled`
- AND the budget reservation is released immediately
