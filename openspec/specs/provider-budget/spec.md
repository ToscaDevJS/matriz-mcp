# Provider and Budget Specification

## Purpose
Defines the generative provider interface, Gemini adapter, offline test fake, pricing lookup, and pre-flight budget enforcement.

## Requirements

### Requirement: Provider Abstraction
The system MUST provide a `Provider` interface with `Name`, `Capabilities`, `EstimateCostUSD`, `Generate`, and `Edit` methods.

#### Scenario: Deterministic fake generation
- GIVEN a `fakeProvider` instance
- WHEN `Generate` is called with the same seed and prompt twice
- THEN it returns identical image bytes and increments its invocation counter

### Requirement: Pre-flight Budget Guard
The system MUST check budget availability via `Guard.Reserve` before any generative request and fail closed if the estimated spend exceeds limits or call count is exhausted.

#### Scenario: Budget ceiling exceeded
- GIVEN a budget limit of $2.00 and $1.90 already spent
- WHEN a request with estimated cost of $0.15 is reserved
- THEN `Reserve` returns an error detailing the limit, spent amount, and estimated cost

#### Scenario: Max generative calls exceeded
- GIVEN a maximum of 20 generative calls and 20 calls completed
- WHEN a 21st call is attempted even with remaining budget
- THEN `Reserve` rejects the call

### Requirement: Closed Cost Estimation
The system MUST estimate costs without network calls and MUST return the worst-case known price for unknown or unlisted models.

#### Scenario: Known model cost estimation
- GIVEN a request for Gemini Flash-Lite at 1K resolution
- WHEN `EstimateCostUSD` is called
- THEN it returns the fixed token-derived cost without making HTTP requests

#### Scenario: Unknown model fallback
- GIVEN a request specifying an unlisted model
- WHEN `EstimateCostUSD` is called
- THEN it returns the worst-case cost in the pricing table rather than 0

### Requirement: Concurrent Multi-draft Generation
When `Generate` is invoked with `Count > 1`, the provider MUST execute concurrent requests to the underlying model in parallel, collect delivered images, offset seeds across workers, and settle budget charges strictly according to the count of delivered images.

#### Scenario: Multi-draft parallel execution
- GIVEN a `GenerateRequest` with `Count = 4`
- WHEN `Generate` is executed
- THEN concurrent requests are dispatched in parallel and up to 4 images are collected without sequential serialization

#### Scenario: Seed distribution across concurrent drafts
- GIVEN a `GenerateRequest` with `Count = 4` and `Seed = 1000`
- WHEN `Generate` dispatches workers $0..3$
- THEN each worker receives a distinct offset seed (`1000, 1001, 1002, 1003`) to ensure distinct visual variations

#### Scenario: Partial failure tolerance
- GIVEN a `GenerateRequest` with `Count = 4` where 1 worker encounters a network failure but 3 succeed
- WHEN `Generate` finishes
- THEN it returns a valid `Result` containing the 3 delivered images and charges the budget for 3 images
