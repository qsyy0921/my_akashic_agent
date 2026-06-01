# SPEC-106: Runtime Overview Media Asset Retention Plan

## Status

Accepted for the current iteration.

## Context

`GET /v1/media-assets/retention-plan` now exposes a Go-owned, read-only dry-run
cleanup plan. Operators still need a single runtime overview entry point that
surfaces whether cleanup is ready, how many candidates exist, and which manual
approval / control mutation audit steps are required.

The Python dashboard currently normalizes media retention diagnostics, but it
does not yet expose the cleanup plan. Python must not own cleanup policy or
execute destructive actions.

## Boundary Analysis

Go owns:

- deterministic retention plan aggregation;
- summary/card/detail fields in `/v1/runtime-overview`;
- stable API shape for cleanup readiness, blockers, candidates, and steps.

Python owns:

- dashboard normalization and display only;
- no deletion, approval creation, mutation audit creation, OCR/VLM, RAG, or AI
  execution.

## Decision

Add runtime overview aggregation for the retention plan:

- Extend `RuntimeOverviewDeps` with a retention planner port.
- Fetch `RetentionPlan(limit)` when available.
- Add `media_asset_retention_plan_*` summary fields.
- Add a `Media Asset Retention Plan` card.
- Include raw `media_asset_retention_plan` detail in the overview response.
- Update Python dashboard fallback to read `/v1/media-assets/retention-plan`
  and normalize Go overview/fallback responses consistently.

All paths remain read-only and return `side_effect=none`.

## Acceptance

- Go service tests assert summary/card/detail.
- Python dashboard tests assert summary/card/detail for Go overview and fallback.
- `go test ./...` and the dashboard pytest target pass.
- SDD TODO is cleared at the end of the iteration.
