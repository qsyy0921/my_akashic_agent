# Review: Phase 8.56 Agent Job Recovery Runner

## Scope

- Added an optional Go runtime background runner for expired `AgentJob` lease
  recovery.
- The runner is disabled by default and does not change existing Python worker
  behavior unless explicitly enabled.
- No platform send path, Python model worker, or NATS acknowledgement path is
  invoked by the runner.

## Design

- `trigger/job` owns the timer/runner adapter.
- `AgentJobService.RecoverExpiredLeases` remains the application use case.
- Domain transition rules remain in `AgentJob.RecoverExpiredLease`.
- Runtime env parsing lives in `cmd/agent-runtime`, keeping process-level
  configuration out of the domain and app layers.

## Configuration

- `AKASHIC_AGENT_JOB_RECOVERY_ENABLED=true`
- `AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS`, default `300`
- `AKASHIC_AGENT_JOB_RECOVERY_LIMIT`, default `50`, max `200`
- `AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START`, default `true` when enabled

## Verification

- `go test ./trigger/job ./cmd/agent-runtime ./app/service -run "TestAgentJobLeaseRecovery|TestAgentJobServiceRecoversExpiredLeases" -count=1 -v`
- `go test ./...`
- `go build ./cmd/agent-runtime`

## Remaining Risk

- Before enabling in a long-running local runtime, do one dry run with current
  Python workers and inspect `lease_expired` events to confirm no unexpected
  heartbeat timing conflict.
