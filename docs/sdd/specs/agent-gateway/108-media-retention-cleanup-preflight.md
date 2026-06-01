# SPEC-108: Media Retention Cleanup Preflight

## Status

Accepted for the current iteration.

## Context

Media asset retention planning now has three read-only building blocks:

- `GET /v1/media-assets/retention-plan` lists cleanup candidates and required
  operator steps.
- control mutation policy allowlists
  `media_asset_retention / cleanup_expired`.
- `GET /v1/control-mutations/preflight` can validate active operator approval
  and return a suggested planned audit.

Operators still need one media-specific preflight endpoint that combines the
cleanup plan with the control mutation preflight. This avoids duplicating the
target/action convention in dashboards or Python code.

## Boundary Analysis

Go owns:

- retention cleanup candidate visibility;
- deterministic approval/preflight gate;
- suggested audit binding metadata;
- stable operator API.

Python owns:

- OCR/VLM/file parsing/semantic extraction;
- no retention cleanup policy, approval, mutation, or deletion logic.

Out of scope:

- deleting media metadata or file content;
- creating approvals or mutation audits automatically;
- executing cleanup;
- calling Python, MQ, RAG, OCR/VLM, or AI.

## Decision

Add a Go app service that:

1. Reads `MediaAssetRetentionPlan`.
2. If no cleanup candidates are ready, returns blocked with the plan reason.
3. If candidates exist, calls `ControlMutationPreflight` with:
   - `target_kind=media_asset_retention`
   - `action=cleanup_expired`
   - caller-provided `target_id`, `operator_id`, and `approval_id`.
4. Returns a single read-only view with plan, preflight, readiness, blockers,
   and `side_effect=none`.

Expose it as:

```text
GET /v1/media-assets/retention-cleanup/preflight
```

## Acceptance

- Service tests cover ready-with-approval, no-candidates, and missing approval.
- HTTP tests cover the endpoint and method guard.
- Full Go regression passes.
- TODO is cleared at the end of the iteration.
