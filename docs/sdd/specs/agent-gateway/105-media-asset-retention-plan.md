# SPEC-105: Go-owned Media Asset Retention Plan

## Status

Accepted for the current iteration.

## Context

Go already owns deterministic media asset registration, content access diagnostics,
and retention diagnostics. Operators can see which assets are past their TTL, but
there is no stable control-plane plan that says whether cleanup is ready, what
would be cleaned, what manual approval should bind to, and how to verify or roll
back.

Media parsing, OCR, VLM, semantic extraction, and file summarization remain AI
runtime concerns in Python. This slice must not delete files or metadata.

## Decision

Add a Go-owned, read-only retention cleanup plan:

- Reuse `MediaAssetService.RetentionDiagnostics` as the source of truth.
- Add `MediaAssetRetentionPlanView` under the query layer.
- Expose `GET /v1/media-assets/retention-plan`.
- Return:
  - `ready`, `reason`, `blockers`
  - `candidate_count`, `asset_count`
  - bounded `candidates`
  - `required_steps`, `verify_steps`, `rollback_steps`
  - `side_effect=none`

The endpoint is a dry-run plan only. It does not create approvals, control
mutation audits, delete metadata, delete files, enqueue jobs, or call Python.

## Boundaries

- Go owns deterministic retention visibility and cleanup planning.
- Python owns OCR/VLM/file parsing/semantic extraction.
- Any future destructive cleanup must be a separate SDD slice and must bind to
  operator approval, control mutation audit, rate limits, and rollback records.

## Acceptance

- Service tests cover due, permanent, and not-due assets.
- HTTP tests cover the new endpoint and `side_effect=none`.
- Full Go regression passes.
- SDD TODO is cleared at the end of the iteration.
