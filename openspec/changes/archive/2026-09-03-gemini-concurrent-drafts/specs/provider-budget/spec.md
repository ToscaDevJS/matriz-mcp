# Provider and Budget Specification (Delta)

## Purpose
Defines generative multi-draft concurrency, partial failure resilience, seed distribution, and settled cost attribution.

## Requirements

### Requirement: Concurrent Multi-draft Generation
When `Generate` is invoked with `Count > 1`, the provider MUST execute concurrent requests to the underlying model in parallel, collect all delivered images, and settle budget charges strictly according to the count of delivered images.

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
