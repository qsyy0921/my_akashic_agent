# SPEC-012: Agent Job Event Stream

## Status

Implemented initial Go-owned durable stream for generic `AgentJob` lifecycle
events.

## Context

Go owns generic `AgentJob` lifecycle state for image generation, media
processing, group memory extraction, RAG ingest, and future eval jobs. The
current repository state is durable when `AKASHIC_AGENT_JOBS_DSN` is configured,
but workers and dashboards still need an append-only lifecycle stream for:

- replaying job transitions after restarts;
- diagnosing lease churn, stale workers, retries, and dead letters;
- preparing a future NATS/Redis Streams adapter without changing Python worker
  contracts.

This is infrastructure lifecycle state, so it belongs in Go.

## Goals

- Record every generic job lifecycle transition as a Go-owned event.
- Keep the existing `/v1/jobs` behavior stable.
- Make event storage pluggable through an outbound port.
- Provide a local JSONL adapter as the first durable stream.
- Expose a read API for tests, dashboard, and operational tooling.

## Non-Goals

- Do not replace the job repository with NATS/Redis Streams in this slice.
- Do not change Python worker lease/result APIs.
- Do not dispatch or execute jobs from the stream directly.

## Domain Model

```text
AgentJobEvent
├── event_id
├── job_id
├── job_type
├── event_type
├── status
├── attempt
├── max_attempts
├── lease_owner
├── lease_expires_at
├── occurred_at
└── metadata
```

Allowed event types:

```text
created
leased
running
succeeded
failed
retry
cancelled
```

The event records the job status after the transition. A `failed` event may
carry `status=dead_lettered` when the failure exhausts attempts.

## Ports

Outbound port:

```text
AgentJobEventStore
├── AppendAgentJobEvent(event)
└── ListAgentJobEvents(filter)
```

Inbound query port:

```text
AgentJobEventViewer
└── List(filter)
```

## Runtime Configuration

```powershell
$env:AKASHIC_AGENT_JOB_EVENTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\agent-job-events.jsonl"
```

`AKASHIC_AGENT_JOB_EVENTS_PATH` is accepted as a shorthand. The special value
`memory` keeps an in-memory development stream.

## HTTP API

```text
GET /v1/job-events?limit=50
GET /v1/job-events?job_id=rag_ingest:qq:3219982:ds1:1
GET /v1/job-events?type=rag_ingest&event=failed
```

The API is read-only. Mutations still happen only through `/v1/jobs`.

Dashboard proxy:

```text
GET /api/dashboard/agent-jobs/{job_id}/events?limit=50
```

The dashboard endpoint reads the Go runtime stream and normalizes the event
fields for plugin panels and contract smoke tests.

## Acceptance

- Creating, leasing, running, succeeding, failing, retrying, and cancelling jobs
  can append lifecycle events.
- The JSONL stream persists events across runtime restarts.
- `/v1/job-events` lists newest events first and supports job/type/event
  filters.
- The dashboard can read a job's event stream without mutating job state.
- Existing job lifecycle tests still pass.
- No Python worker API changes are required.
