# SPEC-014: Runtime Dashboard Overview

## Status

Implemented read-only dashboard plugin with a Go-owned runtime overview
aggregate, runtime config diagnostics, delivery adapter, queue backend,
Go-owned send ledger metrics diagnostics, Go-owned inbox metrics diagnostics,
Go-owned agent job metrics diagnostics, Go-owned outbox metrics diagnostics,
Go-owned runtime worker diagnostics, and Go-owned observe target diagnostics.

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
- Provide an explicit manual adapter health probe for operators, without
  calling live platform APIs during normal overview loading.
- Provide an explicit manual delivery smoke readiness probe backed by Go's
  side-effect-free smoke endpoint.
- Degrade gracefully when one runtime endpoint is unavailable.

## Runtime Reads

The plugin first reads the Go-owned aggregate:

```text
GET /v1/runtime-overview
```

The Go aggregate includes a sanitized runtime config card backed by:

```text
GET /v1/runtime-config
```

This endpoint reports the current process address, address source, bot ids,
selected OneBot/Telegram environment variables, expected OneBot channel aliases,
missing aliases, worker/cutover flags, and `side_effect=none`. Token and secret
values are never returned, including partial token prefixes/suffixes; only
presence plus `redacted` or `channel=redacted` markers are exposed. Boolean
flags whose names contain `TOKEN`, such as `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN`,
remain visible as true/false configuration flags rather than secrets.

When this endpoint is unavailable or returns an invalid shape, the dashboard
falls back to the older multi-endpoint read path:

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
GET /v1/runtime-workers
GET /v1/observe-targets
```

The dashboard also exposes a manual, operator-triggered health proxy:

```text
GET /api/dashboard/runtime-overview/delivery-adapter-health?timeout_seconds=3
```

This proxy calls Go `GET /v1/delivery-adapters/health` only when the user clicks
the `Delivery Adapters` detail action. It is intentionally not part of the
automatic overview read path because it can touch live OneBot/Telegram APIs.

The dashboard also exposes a manual delivery smoke proxy:

```text
GET /api/dashboard/runtime-overview/delivery-smoke-readiness?group_ids=27234224&include_synthetic_media=true
```

This proxy posts to Go `POST /v1/delivery-smoke/readiness` only when the user
clicks the `Delivery Adapters` detail action. It remains side-effect-free:
no outbox row is created and no platform message is sent.

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
- Go-owned runtime worker enabled/running/config state for agent job recovery,
  local outbox dispatch, NATS shadow publish, NATS dual-read compare, and NATS
  external lease cutover.
- Go-owned observe target totals and target metadata for configured
  observe-only QQ groups synced from Python config.
- Go-owned sanitized runtime config state for OneBot aliases, token presence,
  worker flags, and pre-smoke blockers.
- Manual delivery adapter live health results, including reachable,
  authenticated, account id/name, latency, and `side_effect=none`.
- Manual delivery smoke readiness results, including per-case readiness,
  dispatch plan, missing channel aliases, blockers, totals, and
  `side_effect=none`.
- Go-owned aggregate cards and summary fields for runtime overview. Python
  keeps only display normalization and fallback compatibility.

## Boundaries

Go owns:

- authoritative runtime state and lifecycle endpoints;
- job/outbox/checkpoint/event persistence;
- stale lease and dead-letter state.
- send ledger metrics semantics and bounded operational samples.
- inbox metrics semantics and bounded operational samples.
- `agent_job` metrics semantics and bounded operational samples.
- outbox metrics semantics and bounded operational samples.
- runtime worker diagnostics semantics, including enabled/running counters and
  queue worker configuration.
- observe target diagnostics semantics, including source-bound sync, target
  validation, reply-disabled observe-only rules, and `side_effect=none`.
- runtime config diagnostics semantics, including secret redaction, expected
  OneBot alias readiness, and side-effect-free preflight blockers.

Python dashboard owns:

- read-only aggregation and normalization for operator visibility;
- UI rendering;
- endpoint-level fallback/error reporting.

The dashboard must not duplicate Go state transitions. Retry/cancel actions
remain in the specific job/outbox plugins where mutation is explicit.

## Acceptance

- `/api/dashboard/runtime-overview` returns an aggregate summary and cards.
- `/api/dashboard/runtime-overview/delivery-adapter-health` returns normalized
  adapter health only when called explicitly.
- `/api/dashboard/runtime-overview/delivery-smoke-readiness` returns normalized
  delivery smoke readiness only when called explicitly.
- `GET /v1/runtime-overview` returns the Go-owned aggregate summary and cards.
- `GET /v1/runtime-config` returns sanitized runtime configuration with
  `side_effect=none`, no secret leakage, and OneBot alias readiness blockers.
- The panel is discoverable via `/api/dashboard/plugins`.
- Endpoint failures are returned in `status.errors` without breaking the whole
  overview when other endpoints still respond.
- Tests cover summary fields for leases, stale jobs, dead letters, checkpoint
  lag, job events, outbox events, `rag_eval` failures, delivery adapter
  visibility, queue backend visibility, Go-owned send ledger metrics,
  Go-owned inbox metrics, Go-owned `agent_job` metrics, and Go-owned outbox
  metrics, and Go-owned runtime worker diagnostics.
- Tests cover Python fallback to the old multi-endpoint path when the Go
  aggregate is unavailable.
- Tests cover the manual adapter health proxy and confirm normal overview
  loading does not call the health endpoint.
- Tests cover the manual delivery smoke proxy and confirm the frontend exposes
  the manual action.
- Tests cover runtime config endpoint output, secret redaction, missing OneBot
  alias blockers, and runtime overview Runtime Config card status.
