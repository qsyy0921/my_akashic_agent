# AgentJob Priority Plan

## Context

`AgentJob Capacity Plan` can explain worker recovery and concurrency tuning, but
operators still need a stable read-only view of which job types should be
handled first when backlog appears. This belongs in Go because it is
deterministic runtime control-plane logic derived from job state and worker
heartbeats.

Python remains the AI execution layer for model calls, prompts, OCR/VLM, image
generation, group memory, RAG ingestion, rerank, evaluation, and provider
experiments.

## Goals

- Add a Go-owned read-only priority/backpressure plan for generic AgentJobs.
- Derive priority order from existing AgentJob pressure and Python worker
  coverage.
- Expose the plan through HTTP without mutating job, queue, worker, or config
  state.
- Keep the output suitable for future operator acknowledgement, MQ priority, or
  worker concurrency work.

## Non-Goals

- No automatic scheduling or priority mutation.
- No worker startup, autoscaling, or concurrency control.
- No AgentJob lease, ack, nack, term, retry, create, cancel, or result writeback.
- No Python AI execution.
- No provider-specific RAGFlow or model behavior.

## API

`GET /v1/agent-job-priority/plan`

Query parameters:

- `job_limit`: bounded list size for job metrics.
- `event_limit`: bounded list size for event metrics.
- `stale_after_seconds`: Python worker heartbeat staleness threshold.

Response data:

- `ready`: whether priority advice can be trusted without blockers.
- `reason`: stable reason string.
- `summary`: total job types, high-priority job types, blocked job types, and
  maximum priority score.
- `items`: ordered per-job-type recommendations.
- `verification_steps`: manual checks before any future cutover.
- `blockers`: runtime issues that prevent trusting or acting on the plan.
- `attributes.side_effect=none` and `side_effect=none`.

## Priority Rules

The first version is deliberately deterministic and conservative:

- high pressure with no active mapped worker: `critical`, score `100`;
- failed worker present: `high`, score `90`;
- stale worker present: `high`, score `80`;
- high pressure with active worker: `medium`, score `70`;
- mapped job type with no active worker: `medium`, score `60`;
- unmapped job type: `advisory`, score `10`;
- healthy covered job type: `low`, score `20`.

Ties are sorted by pending count, oldest pending age, active count, then job type.

## Acceptance

- Unit tests prove ordering, blockers, severity, and read-only attributes.
- HTTP tests prove the endpoint returns UTF-8 JSON and rejects non-GET methods.
- Existing runtime tests remain green.

## Review Notes

- This is tactical DDD/hexagonal: command/query types in `app`, use case in
  `app/service`, inbound port in `app/port/in`, HTTP adapter in `trigger/http`,
  wiring in `cmd/agent-runtime`.
- It is a control-plane read model, not a scheduler.
