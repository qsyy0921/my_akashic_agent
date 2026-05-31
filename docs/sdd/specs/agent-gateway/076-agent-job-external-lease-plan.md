# 076 Agent Job External Lease Plan

## Context

`/v1/agent-job-external-lease/readiness` explains whether generic `agent_job`
NATS external lease result-ack is safe. Operators still need a stable,
repeatable plan that lists required checks, environment changes, verification
steps and rollback steps before changing runtime ownership.

This is a Go runtime control-plane concern. Python remains the AI worker that
leases and executes image generation, memory/RAG extraction, OCR/VLM,
provider-specific fallback and prompt/tool pipelines.

## Decision

Add:

```text
GET /v1/agent-job-external-lease/plan
```

The endpoint accepts optional query params:

- `desired_execution_owner`;
- `job_limit`;
- `event_limit`;
- `stale_after_seconds`.

It returns a read-only plan containing:

- `ready`;
- `decision`;
- `desired_execution_owner`;
- `recommended_execution_owner`;
- `current_execution_owner`;
- embedded readiness from `/v1/agent-job-external-lease/readiness`;
- `required_checks`;
- `enable_steps`;
- `verification_steps`;
- `rollback_steps`;
- `blockers`;
- `side_effect=none`.

The initial supported desired owner is
`python_ai_worker_with_nats_result_ack`. The current fallback owner is
`python_ai_worker_state_store_lease`.

## Boundary

### Go owns

- plan generation and blocker classification;
- result-ack cutover checklist;
- explicit environment variable hints and verification endpoints;
- rollback instructions.

### Python owns

- all actual AI job execution;
- model/provider calls, prompt/tool routing, memory/RAG algorithms and
  multimodal processing;
- worker heartbeat reporting and AgentJob lease execution.

## Non-goals

- No AgentJob lease, renew, complete, fail or retry.
- No NATS ack/nack/term.
- No worker startup.
- No environment mutation.
- No model/provider/tool execution.

## Validation

- Service tests cover ready and blocked plans.
- HTTP tests cover the read-only endpoint.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
