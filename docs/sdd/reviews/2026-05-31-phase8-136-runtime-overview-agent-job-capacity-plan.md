# Phase 8.136 Runtime Overview AgentJob Capacity Plan

Spec: `docs/sdd/specs/agent-gateway/079-runtime-overview-agent-job-capacity-plan.md`

## Changes

- Added `AgentJobCapacityPlan` to `RuntimeOverviewView`.
- Added `RuntimeOverviewDeps.AgentJobCapacityPlan`.
- Wired `agentJobCapacityPlan` into `cmd/agent-runtime`.
- Added runtime overview summary fields for capacity ready/reason/blockers,
  high-pressure job types, blocked job types, worker warning job types, active
  worker job types, and max pending/oldest pending age.
- Added `Agent Job Capacity` runtime overview card with full plan detail.

## Boundary Review

- Go only aggregates the read-only plan for dashboard/control-plane visibility.
- Python still owns actual worker lifecycle, concurrency changes, model calls,
  memory/RAG/OCR/VLM/image execution, prompt and tool work.
- No scheduling, autoscaling, job leasing, retries, MQ acknowledgement, config
  mutation or operator acknowledgement was added in this slice.

## Verification

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `go test ./trigger/http -run TestRuntimeOverviewEndpointReturnsGoOwnedAggregate -count=1 -v`
- `go test ./cmd/agent-runtime -run TestRuntimeConfig -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Residual Risk

- Runtime overview now depends on the capacity planner when wired. If that
  planner fails, runtime overview reports a partial error and omits the card
  detail rather than mutating runtime state.
- Live correctness still depends on Python workers reporting accurate
  `agent-worker-statuses`.
