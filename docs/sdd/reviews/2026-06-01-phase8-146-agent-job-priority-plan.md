# Phase 8.146 AgentJob Priority Plan

Spec: `docs/sdd/specs/agent-gateway/089-agent-job-priority-plan.md`

## Summary

- Added Go `GET /v1/agent-job-priority/plan`.
- Added `AgentJobPriorityPlanService` under `services/agent-runtime/app/service`.
- Added command/query/inbound-port types for a read-only priority/backpressure
  plan.
- Wired the endpoint in `cmd/agent-runtime`.

## Boundary Review

This slice keeps Go in the deterministic runtime/control-plane role. The plan
only reads AgentJob pressure and Python worker heartbeats, then returns a stable
ranked recommendation list.

It does not:

- create, lease, retry, cancel, complete, or fail AgentJobs;
- publish, ack, nack, term, or lease MQ work;
- start Python workers or change worker concurrency;
- execute model, prompt, OCR/VLM, image generation, memory, RAG, or eval logic;
- mutate runtime configuration.

Python remains the execution owner for AI work.

## Tests

- `go test ./app/service -run TestAgentJobPriorityPlan -count=1 -v`
- `go test ./trigger/http -run TestAgentJobPriorityPlanEndpointReturnsReadOnlyPlan -count=1 -v`
- `go test ./...`

## Risks

- Priority scores are intentionally conservative heuristics. They are suitable
  for operator visibility and future cutover planning, not for automatic
  scheduling yet.
- Real priority scheduling still needs operator acknowledgement, config audit,
  queue adapter support, and Python worker concurrency implementation.

## Decision

Accept as a read-only control-plane slice. Do not promote it to automatic
priority scheduling without a separate SDD and rollback plan.
