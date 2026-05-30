# Review: Phase 8.57 Agent Job NATS Flow Smoke

## Scope

- Added a local NATS JetStream smoke for `agent_job` result acknowledgement.
- The smoke verifies one job across `pending`, `running`, and `succeeded`
  states without invoking Python workers or platform delivery adapters.
- Added `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED=true` as an explicit gate
  before the live external lease consumer expands from outbox-only subjects to
  `agent_job` result-ack subjects.

## Design

- Go still treats the `AgentJob` state store as authoritative.
- `pending` queue notifications delayed-`nack` so Python can lease and execute.
- `running` queue notifications delayed-`nack` while waiting for Python result
  writeback.
- `succeeded` queue notifications `ack` after terminal state is visible in Go.
- The new gate complements the duplicate terminal delivery smoke; both must pass
  before live `agent_job` subject consumption is allowed.

## Verification

- `AKASHIC_NATS_SMOKE_DSN=nats://127.0.0.1:4222 go test ./smoke -run TestExternalLeaseNATSSmokeAgentJobPendingRunningSucceededFlow -count=1 -v`

## Remaining Risk

- The smoke uses Go service calls to simulate Python lease/running/succeeded
  writeback. Before live cutover, run a full local runtime dry-run with the real
  Python worker process and strict lease-token mode enabled.
