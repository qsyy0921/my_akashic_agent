# SPEC-111: Dashboard Media Retention Cleanup Detail

## Status

Accepted for the current iteration.

## Context

Go runtime now exposes `media_asset_retention_cleanup` in
`/v1/runtime-overview`. Python dashboard currently normalizes media retention
diagnostics and plan, but it does not normalize the cleanup aggregate. Without a
dashboard adapter, frontend users can miss cleanup readiness and recent
metadata-only cleanup audits even though Go owns the control-plane data.

## Boundary Analysis

Go owns:

- retention cleanup plan/preflight/execution state;
- cleanup audit aggregation;
- stable runtime overview source of truth.

Python owns:

- dashboard read-only presentation;
- fallback defaults when Go runtime is unavailable.

Out of scope:

- executing cleanup from dashboard;
- creating approvals or mutation audits;
- deleting metadata or files;
- OCR/VLM/RAG/AI calls.

## Decision

Update the runtime overview dashboard adapter to:

- normalize `media_asset_retention_cleanup`;
- provide summary defaults for cleanup readiness, candidates, blockers, audit
  counts, and recent audit count;
- include a dashboard card when the Go card exists or when fallback cards are
  generated;
- keep `side_effect=none`.

## Acceptance

- `tests/test_runtime_overview_dashboard_plugin.py` covers summary/card/detail.
- Fallback path exposes stable cleanup defaults.
- Target Python test passes.
- Go regression still passes.
- TODO is cleared at the end of the iteration.
