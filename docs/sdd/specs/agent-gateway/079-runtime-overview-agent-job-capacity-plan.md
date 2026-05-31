# 079 Runtime Overview AgentJob Capacity Plan

Date: 2026-05-31

## Problem

`GET /v1/agent-job-capacity/plan` now turns AgentJob pressure plus Python
worker coverage into read-only operational recommendations. Operators and the
dashboard still need a single runtime entrypoint that surfaces the same capacity
state next to queue ownership, AgentJob pressure, worker coverage, external
lease gates and knowledge pipeline diagnostics.

Without aggregation, the dashboard has to know another specialized endpoint and
operators can miss capacity blockers during observe-only QQ group knowledge
ingestion, image generation, RAG ingest, or NATS result-ack preparation.

## Scope

- Add `AgentJobCapacityPlan` to `RuntimeOverviewView`.
- Add `RuntimeOverviewDeps.AgentJobCapacityPlan`.
- Call the planner with the same bounded `limit`, `event_limit` and
  `stale_after_seconds` used by other runtime overview diagnostics.
- Add summary fields:
  - `agent_job_capacity_ready`
  - `agent_job_capacity_reason`
  - `agent_job_capacity_blockers`
  - `agent_job_capacity_high_pressure_job_types`
  - `agent_job_capacity_blocked_job_types`
  - `agent_job_capacity_worker_warning_job_types`
  - `agent_job_capacity_active_worker_job_types`
  - `agent_job_capacity_max_pending`
  - `agent_job_capacity_oldest_pending_age_seconds`
- Add `Agent Job Capacity` card with full plan detail.

## Go / Python Boundary

Go owns:

- runtime-overview aggregation;
- stable API shape for dashboard consumption;
- deterministic card status/value and summary fields;
- SDD and tests.

Python owns:

- actual AI worker lifecycle and concurrency changes;
- model calls, prompt/tool work, memory/RAG/OCR/VLM/image generation;
- future strategy experiments behind the worker implementation.

## Non-goals

- No worker startup.
- No autoscaling.
- No job lease/retry.
- No MQ acknowledgement.
- No config mutation or operator ack.
- No dashboard UI redesign in this slice.

## API

```text
GET /v1/runtime-overview?limit=200&event_limit=50&stale_after_seconds=900
```

New response fields:

```text
{
  "summary": {
    "agent_job_capacity_ready": false,
    "agent_job_capacity_reason": "agent_job_capacity_attention_required",
    "agent_job_capacity_blockers": 3,
    "agent_job_capacity_blocked_job_types": 1
  },
  "cards": [
    {
      "id": "agent_job_capacity_plan",
      "label": "Agent Job Capacity",
      "value": "attention:3",
      "status": "danger",
      "detail": {
        "agent_job_capacity_plan": {}
      }
    }
  ],
  "agent_job_capacity_plan": {}
}
```

## Acceptance

- Runtime overview service test verifies summary, card and detail fields.
- HTTP runtime overview route test can return the new card/summary shape.
- `cmd/agent-runtime` wires the existing `agentJobCapacityPlan`.
- README documents the runtime overview aggregation.
- Review doc records tests and residual risks.
