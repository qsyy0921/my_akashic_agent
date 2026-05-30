# SPEC-014: Runtime Dashboard Overview

## Status

Implemented read-only dashboard plugin with delivery adapter and queue backend
diagnostics.

## Context

The Go `agent-runtime` now owns several infrastructure control planes:
outbox, generic agent jobs, knowledge checkpoints, knowledge worker
diagnostics, media metadata, and job lifecycle events. Existing dashboard
plugins expose each plane independently, but migration work needs a single
operational overview to catch stale leases, dead letters, checkpoint lag, and
runtime availability before cutting more responsibilities over to Go.

## Goals

- Add a dashboard panel that aggregates Go runtime health and operational
  state.
- Reuse existing runtime endpoints instead of adding a new Go API in this
  slice.
- Keep the panel read-only; it must not send QQ/Telegram messages or mutate
  jobs.
- Degrade gracefully when one runtime endpoint is unavailable.

## Runtime Reads

The plugin reads:

```text
GET /healthz
GET /v1/jobs
GET /v1/outbox
GET /v1/knowledge-checkpoints
GET /v1/knowledge-worker-diagnostics
GET /v1/job-events
GET /v1/outbox-events
GET /v1/delivery-adapters
GET /v1/queue-backend
```

It summarizes:

- runtime health and endpoint errors;
- active worker leases across generic jobs and outbox dispatch;
- stale jobs using knowledge diagnostics plus computed lease expiry;
- dead-lettered jobs and outbox deliveries;
- maximum checkpoint lag from `latest_source_seq - cursor`;
- recent job lifecycle event count;
- recent outbox lifecycle event count;
- `rag_eval` quality failures.
- configured delivery adapter count, enabled count, and disabled count.
- current MQ provider/mode, consumer concurrency, max-in-flight, and external
  lease gate readiness.

## Boundaries

Go owns:

- authoritative runtime state and lifecycle endpoints;
- job/outbox/checkpoint/event persistence;
- stale lease and dead-letter state.

Python dashboard owns:

- read-only aggregation for operator visibility;
- UI rendering;
- endpoint-level fallback/error reporting.

The dashboard must not duplicate Go state transitions. Retry/cancel actions
remain in the specific job/outbox plugins where mutation is explicit.

## Acceptance

- `/api/dashboard/runtime-overview` returns an aggregate summary and cards.
- The panel is discoverable via `/api/dashboard/plugins`.
- Endpoint failures are returned in `status.errors` without breaking the whole
  overview when other endpoints still respond.
- Tests cover summary fields for leases, stale jobs, dead letters, checkpoint
  lag, job events, outbox events, `rag_eval` failures, delivery adapter
  visibility, and queue backend visibility.
