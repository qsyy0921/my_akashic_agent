# 073 Runtime Overview Outbound Cutover Plan

## Context

`/v1/outbound-cutover/readiness` and `/v1/outbound-cutover/plan` are now
available as direct operator APIs, but the dashboard entrypoint is
`/v1/runtime-overview`. Without aggregation, operators must know to call the
specialized cutover endpoint before changing QQ/NapCat delivery ownership.

The runtime overview should surface the same read-only plan in summary/card
form so the current execution owner, recommended owner, blockers and rollback
availability are visible beside queue, worker and adapter diagnostics.

## Decision

Add an optional outbound cutover plan dependency to `RuntimeOverviewService`.

When present, runtime overview calls:

```text
PlanOutboundCutover(command.PlanOutboundCutoverCommand{})
```

This uses the default smoke matrix and `desired_execution_owner=auto`, matching
the operator-facing plan endpoint. The overview then returns:

- `outbound_cutover_plan_ready`;
- `outbound_cutover_plan_decision`;
- `outbound_cutover_plan_blockers`;
- `outbound_cutover_plan_current_owner`;
- `outbound_cutover_plan_recommended_owner`;
- `outbound_cutover_plan_desired_owner`;
- an `outbound_cutover_plan` card with detail.

## Boundary

### Go owns

- overview aggregation;
- read-only plan invocation;
- summary/card shape for dashboard/operator APIs.

### Python owns

- AI content creation and model/tool execution;
- compatibility sending before explicit Go cutover;
- provider-specific AI behavior.

## Non-goals

- No platform sends.
- No environment mutation.
- No worker start/stop.
- No new dashboard UI in this slice.

## Validation

- Runtime overview service tests cover summary/card/detail.
- `go test ./...`, `go build ./cmd/agent-runtime`, and `git diff --check`
  must pass.
