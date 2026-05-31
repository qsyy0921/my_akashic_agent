# 078 AgentJob Capacity Plan

Date: 2026-05-31

## Problem

`/v1/job-metrics` can show `AgentJob` pressure, and `/v1/runtime-overview`
can correlate pressure with Python worker coverage. The next operational need is
a read-only plan that answers: which job types need worker recovery, worker
restart, or concurrency tuning before introducing any real autoscaling or
priority scheduling.

Without a dedicated plan endpoint, operators must infer capacity actions from
multiple diagnostic payloads. That is error-prone during observe-only QQ group
knowledge ingestion, image generation, RAG ingest, and future NATS result-ack
cutover.

## Scope

- Add `GET /v1/agent-job-capacity/plan`.
- Aggregate Go-owned `AgentJob` pressure and Python `agent-worker-statuses`.
- Reuse the existing `AgentJobWorkerCoverageView` mapping.
- Return per-job-type action recommendations and a summary.
- Keep the endpoint read-only: no job lease, no retry, no worker startup, no MQ
  acknowledgement, no config mutation and no AI execution.

## Go / Python Boundary

Go owns:

- bounded job/event pressure aggregation;
- worker status coverage interpretation;
- deterministic recommendation labels and operational verification hints;
- JSON API, tests, SDD and audit-facing docs.

Python owns:

- actual AI worker process lifecycle;
- model calls, prompt/tool work, group memory extraction, RAG/chunking,
  embedding/rerank, OCR/VLM and image generation;
- any future worker concurrency implementation or strategy experiment.

## API

```text
GET /v1/agent-job-capacity/plan?job_limit=200&event_limit=50&stale_after_seconds=900
```

Response data:

```text
{
  "ready": false,
  "reason": "agent_job_capacity_attention_required",
  "summary": {
    "job_types": 3,
    "high_pressure_job_types": 2,
    "capacity_blocked_job_types": 1,
    "worker_warning_job_types": 1,
    "max_pending": 10,
    "max_active": 5,
    "oldest_pending_age_seconds": 1200
  },
  "items": [
    {
      "job_type": "group_memory_extract",
      "severity": "danger",
      "action": "start_or_recover_python_worker",
      "recommendation": "Start or recover a Python knowledge worker before enabling more admission.",
      "coverage": {}
    }
  ],
  "verification_steps": [],
  "side_effect": "none"
}
```

## Recommendation Rules

- `danger/start_or_recover_python_worker`: high pressure and no active worker.
- `warn/inspect_failed_worker`: failed worker exists.
- `warn/renew_or_restart_stale_worker`: stale worker exists.
- `warn/increase_worker_concurrency_or_prioritize_queue`: high pressure with at
  least one active worker.
- `ok/monitor`: active worker exists and no high pressure or worker warning.
- `muted/no_worker_mapping`: job type has no known Python worker mapping.

The plan is intentionally advisory. It does not decide exact concurrency,
replica count, or priority weights.

## DDD / Hexagonal Placement

- `app/command`: request parameters.
- `app/query`: plan DTOs.
- `app/port/in`: `AgentJobCapacityPlanner`.
- `app/service`: application service composing existing metrics and worker
  status ports.
- `trigger/http`: GET route and query parsing.
- `cmd/agent-runtime`: wiring only.

## Acceptance

- Service tests cover danger, failed/stale/high-pressure-active and read-only
  metadata.
- HTTP route test covers `GET /v1/agent-job-capacity/plan`.
- README documents the endpoint and non-goals.
- SDD TODO is cleared only after tests, review docs, commit and push.
