# Review: Runtime Dashboard Overview

Spec: `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

## Implementation Summary

- Added `plugins/runtime_overview` as a read-only dashboard plugin.
- Aggregates `/healthz`, `/v1/jobs`, `/v1/outbox`,
  `/v1/knowledge-checkpoints`, `/v1/knowledge-worker-diagnostics`, and
  `/v1/job-events`.
- Exposes `/api/dashboard/runtime-overview` with summary cards for runtime
  health, worker leases, stale jobs, dead letters, checkpoint lag, event stream
  activity, and `rag_eval` failures.
- Added a compact panel UI with TypeScript source, generated browser JS, and
  plugin CSS.

## Tests Run

- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp/pytest-runtime-overview`

## Findings

- The plugin is intentionally read-only and does not mutate jobs or trigger
  platform sends.
- Stale detection combines Go knowledge diagnostics with local lease expiry
  computation, so the overview remains useful even when diagnostic details are
  partial.

## Decision

Approved for this migration slice. This improves operational visibility before
moving more runtime infrastructure responsibilities to Go.

## Follow-ups

- Add a dedicated `rag_eval` trend and quality-gate panel.
- Consider moving aggregate overview into a Go-owned endpoint after the shape is
  stable.
