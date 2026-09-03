# Tasks: Remove AVIF Codec & Streamline Codecs

## Phase 1: Core Pipeline & Codec Clean-up (TDD)
- [x] 1.1 Update `internal/core/encode_test.go` to test AVIF rejection and assert JPEG, PNG, WebP encoding (RED).
- [x] 1.2 Update `internal/core/encode.go`: remove `gen2brain/avif` import, update `ParseFormat` to reject AVIF with guidance, update `Encode` and remove `FormatAVIF` from active encoders (GREEN).
- [x] 1.3 Prune `gen2brain/avif` from `go.mod` and run `go mod tidy`.

## Phase 2: Doctor Diagnostics Alignment
- [x] 2.1 Update `internal/doctor/doctor_test.go` and `internal/doctor/probes_codecs.go` to probe PNG, JPEG, WebP.
- [x] 2.2 Verify `matriz doctor` runs clean.

## Phase 3: Documentation & Spec Synchronization
- [x] 3.1 Update `openspec/specs/core-pipeline/spec.md`.
- [x] 3.2 Update `README.md`, `PENDING.md` (mark item #4 resolved), and `CHANGELOG.md`.
- [x] 3.3 Verify full test suite passes with `go test -v -count=1 ./...` and `go build ./...`.
