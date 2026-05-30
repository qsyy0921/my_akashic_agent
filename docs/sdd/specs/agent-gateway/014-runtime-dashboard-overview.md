# SPEC-014: Runtime Dashboard Overview

## Status

Implemented read-only dashboard plugin with delivery adapter, queue backend,
Go-owned send ledger metrics diagnostics, Go-owned inbox metrics diagnostics,
Go-owned agent job metrics diagnostics, and Go-owned outbox metrics diagnostics.

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
- Prefer Go-owned aggregate endpoints when the runtime owns the metric
  semantics; Python only normalizes for display.
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
GET /v1/send-ledger/metrics
GET /v1/inbox-metrics
GET /v1/job-metrics
GET /v1/outbox-metrics
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
- Go-owned send ledger bot/conversation coverage and repeated content hash
  risk for loop guard audits.
- Go-owned inbox observe-only, attachment capture, conversation, sender, and
  sequence cursor metrics.
- Go-owned `agent_job` lifecycle throughput and dead-letter trend samples.
- Go-owned outbox delivery lifecycle throughput and dead-letter trend samples.

## Boundaries

Go owns:

- authoritative runtime state and lifecycle endpoints;
- job/outbox/checkpoint/event persistence;
- stale lease and dead-letter state.
- send ledger metrics semantics and bounded operational samples.
- inbox metrics semantics and bounded operational samples.
- `agent_job` metrics semantics and bounded operational samples.
- outbox metrics semantics and bounded operational samples.

Python dashboard owns:

- read-only aggregation and normalization for operator visibility;
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
  visibility, queue backend visibility, Go-owned send ledger metrics,
  Go-owned inbox metrics, Go-owned `agent_job` metrics, and Go-owned outbox
  metrics.
