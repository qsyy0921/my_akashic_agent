# 075 Runtime Overview Agent Job External Lease Readiness

## Context

`/v1/agent-job-external-lease/readiness` now provides a focused read-only gate
for moving generic `agent_job` queue notifications into NATS external lease
result-ack. The dashboard and operator entrypoint is still
`/v1/runtime-overview`, so the readiness signal should be visible next to queue
backend, AgentJob pressure, Python worker coverage and runtime config.

This is a Go runtime control-plane concern. Python continues to execute all AI
jobs and only reports worker status/heartbeats into Go.

## Decision

Add an optional `AgentJobExternalLeaseReady` dependency to
`RuntimeOverviewService`.

When present, runtime overview calls:

```text
CheckAgentJobExternalLeaseReadiness(command.CheckAgentJobExternalLeaseReadinessCommand{
  JobLimit: filter.Limit,
  EventLimit: filter.EventLimit,
  StaleAfterSeconds: filter.StaleAfterSeconds,
})
```

The overview returns:

- `agent_job_external_lease_ready`;
- `agent_job_external_lease_reason`;
- `agent_job_external_lease_blockers`;
- `agent_job_external_lease_result_ack_ready`;
- `agent_job_external_lease_worker_ready`;
- `agent_job_external_lease_strict_token`;
- `agent_job_external_lease_execution_owner`;
- `agent_job_external_lease_execution_scope`;
- an `agent_job_external_lease_readiness` card with full readiness detail.

## Boundary

### Go owns

- overview aggregation;
- readiness summary/card shape for dashboard/operator APIs;
- read-only correlation of queue gate, strict tokens, job pressure and worker
  coverage.

### Python owns

- all model/provider, prompt/tool, memory/RAG, image/OCR/VLM execution;
- worker status reporting into Go.

## Non-goals

- No AgentJob lease, renew, complete, fail or retry.
- No NATS ack/nack/term.
- No Python worker startup.
- No environment mutation.

## Validation

- Runtime overview tests cover summary/card/detail.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
