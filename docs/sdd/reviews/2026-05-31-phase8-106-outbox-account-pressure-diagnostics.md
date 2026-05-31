# Phase 8.106 Review: Outbox Account Pressure Diagnostics

## Spec

- `docs/sdd/specs/agent-gateway/049-outbox-account-pressure-diagnostics.md`
- `docs/sdd/specs/agent-architecture/003-python-go-boundary.md`
- `docs/sdd/ITERATION_PROMPT.md`

## Implementation Summary

- Added Go-owned `pressure` diagnostics to `OutboxMetricsView`.
- Outbox metrics now aggregate queued, dispatching, active, and dead-lettered
  delivery counts per `channel_kind:account_id`.
- Runtime overview now exposes `outbox_pressure_*` summary values and an
  `outbox_pressure` card.
- Expanded SDD process/prompt/boundary docs so Python responsibilities are
  explicit: model/provider routing, prompt/context, tools, Memory/RAG
  algorithms, chunking, retrieval, embedding/rerank, OCR/VLM, image generation,
  group-knowledge distillation, evaluation, and provider-specific experiments.
- Reaffirmed that each iteration must complete all current `TODO.md` items
  before stopping.

## Tests Run

- `go test ./app/service -run "TestOutboxMetricsService|TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics" -count=1 -v`
- `go test ./...`

## Findings

- No code-review blockers found in the implemented scope.
- The new pressure thresholds are intentionally constants for read-only
  diagnostics. They do not pause workers, reject enqueues, or change retry
  behavior.
- Python is not involved in this diagnostic slice, which keeps AI logic and
  runtime infrastructure separated.

## Decision

Approved as a read-only Go runtime diagnostic. It is suitable groundwork for a
future account-level rate-limit/backpressure slice, but it is not that
enforcement layer.

## Follow-ups

- Live observe `/v1/outbox-metrics` and `/v1/runtime-overview` after artificial
  or real account-level backlog.
- Design account-level send limit policy separately before enabling any
  production blocking behavior.
