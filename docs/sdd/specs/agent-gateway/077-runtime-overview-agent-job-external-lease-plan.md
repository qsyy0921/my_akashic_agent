# 077 Runtime Overview Agent Job External Lease Plan

## Context

`/v1/agent-job-external-lease/plan` now provides the read-only operator plan for
moving generic `agent_job` queue acknowledgement into NATS external lease
result-ack. The main dashboard/operator entrypoint is still
`/v1/runtime-overview`, so the plan should be visible next to AgentJob pressure,
Python worker coverage, external-lease readiness and outbound cutover state.

This is a Go runtime control-plane concern. Python continues to execute all AI
jobs and only reports job/worker status back to Go.

## Decision

Add an optional `AgentJobExternalLeasePlan` dependency to
`RuntimeOverviewService`.

When present, runtime overview calls:

```text
PlanAgentJobExternalLease(command.PlanAgentJobExternalLeaseCommand{
  Readiness: command.CheckAgentJobExternalLeaseReadinessCommand{
    JobLimit: filter.Limit,
    EventLimit: filter.EventLimit,
    StaleAfterSeconds: filter.StaleAfterSeconds,
  },
})
```

The overview returns:

- `agent_job_external_lease_plan_ready`;
- `agent_job_external_lease_plan_decision`;
- `agent_job_external_lease_plan_blockers`;
- `agent_job_external_lease_plan_current_owner`;
- `agent_job_external_lease_plan_desired_owner`;
- `agent_job_external_lease_plan_recommended_owner`;
- an `agent_job_external_lease_plan` card with full plan detail.

## Boundary

### Go owns

- overview aggregation;
- read-only plan summary/card/detail for dashboard/operator APIs;
- correlation of result-ack plan, queue ownership and Python worker coverage
  readiness.

### Python owns

- all model/provider, prompt/tool, memory/RAG, image/OCR/VLM execution;
- worker status reporting and AgentJob execution.

## Non-goals

- No AgentJob lease, renew, complete, fail or retry.
- No NATS ack/nack/term.
- No Python worker startup.
- No environment mutation.
- No model/tool/provider execution.

## Validation

- Runtime overview tests cover summary/card/detail.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
