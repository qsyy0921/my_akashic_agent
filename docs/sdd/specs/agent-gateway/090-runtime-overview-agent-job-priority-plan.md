# Runtime Overview AgentJob Priority Plan

## Context

`/v1/agent-job-priority/plan` now exposes a Go-owned read-only priority and
backpressure plan for generic AgentJobs. Operators normally inspect runtime
state through `/v1/runtime-overview` and the Python dashboard proxy, so the plan
must be included there as a first-class control-plane detail.

## Ownership Boundary

Go owns:

- deterministic aggregation of AgentJob priority plan state;
- runtime overview summary/card/detail fields;
- the stable API contract consumed by dashboards.

Python owns:

- AI worker execution;
- model calls, prompts, OCR/VLM, image generation, memory/RAG algorithms;
- dashboard read-only normalization and presentation.

## Goals

- Add optional `AgentJobPriorityPlan` dependency to `RuntimeOverviewService`.
- Add runtime summary fields for readiness, reason, blockers, job types, high
  priority job types, blocked/warning job types and max priority score.
- Add `Agent Job Priority` card with full plan detail.
- Normalize `agent_job_priority_plan` in the Python dashboard aggregate path.
- Keep the change read-only and side-effect free.

## Non-Goals

- No automatic priority scheduling.
- No MQ priority mutation, ack/nack, lease, retry or result writeback.
- No worker startup, autoscaling, or concurrency mutation.
- No Python AI execution.

## Acceptance

- Go tests prove runtime overview summary/card/detail include the priority plan.
- Python dashboard tests prove the Go aggregate is normalized with stable default
  summary fields and detail shape.
- Existing Go and Python dashboard tests remain green.
