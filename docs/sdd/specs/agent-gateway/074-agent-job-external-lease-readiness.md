# 074 Agent Job External Lease Readiness

## Context

NATS `external_lease` can already execute outbox delivery after explicit
cutover gates. Generic `agent_job` is different: Go owns lifecycle, lease token,
retry, recovery and result-ack mapping, but Python still owns actual AI work
such as image generation, group memory extraction, RAG ingest, RAG eval,
OCR/VLM, prompt execution and provider fallback.

Before moving `agent_job` queue subjects into the NATS external lease consumer,
operators need a single read-only gate that explains whether result-ack is
safe. `/v1/queue-backend` exposes raw gate fields, while runtime overview
exposes worker pressure. This readiness endpoint combines those signals into a
specific answer.

## Decision

Add:

```text
GET /v1/agent-job-external-lease/readiness
```

The endpoint aggregates:

- queue backend provider/mode/external lease gate;
- whether `agent_job` is included in `allowed_work_kinds`;
- whether the queue execution owner is `python_ai_worker_with_nats_result_ack`;
- runtime strict lease-token config;
- current Python worker coverage for pressured AgentJob types.

Readiness is true only when:

- external lease execution is allowed;
- `agent_job` result-ack scope is allowed;
- strict AgentJob lease-token mode is enabled;
- runtime config says external lease agent-job result-ack is enabled;
- current pressured job types do not have danger-level Python worker coverage.

## Boundary

### Go owns

- readiness aggregation and blocker classification;
- queue result-ack gate visibility;
- worker coverage correlation from Go-owned job pressure and worker status.

### Python owns

- all actual AI job execution;
- model/provider calls, prompt/tool execution, RAG/memory algorithms and
  multimodal processing;
- worker heartbeat reporting to Go runtime.

## Non-goals

- No NATS ack/nack is performed.
- No AgentJob is leased, renewed, completed, failed or retried.
- No Python worker is started.
- No env var is mutated.
- No platform message is sent.

## Validation

- Service tests cover ready result-ack and blocked worker coverage.
- HTTP tests cover read-only endpoint behavior.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
