# Review: Phase 8.54 Agent Job Result-Ack Mapping

## Scope

- Added safe `agent_job` disposition mapping to the Go external lease executor.
- This slice does not let Go execute Python-owned model, RAG, OCR/VLM, or memory
  work.
- Live runtime scope remains `outbox_delivery_only`; NATS `agent_job` subject
  cutover still requires a separate smoke and explicit scope expansion.

## Design

- `WorkQueueExternalLeaseService` can now receive an `AgentJobService`
  dependency.
- For `agent_job` queue notifications, Go reads the `AgentJob` aggregate and
  returns a queue disposition only.
- Go never leases an `agent_job` on behalf of Python in this slice, avoiding the
  deadlock where Python state-store workers can no longer acquire the job.

## Disposition Rules

- `pending`: delayed `nack`, because Python still needs to lease and execute.
- active `leased` / `running`: delayed `nack`, because result writeback has not
  happened yet.
- `failed`: Go retries through the domain retry path, then delayed `nack`.
- expired active lease: Go recovers the lease first; retryable jobs delayed
  `nack`, exhausted jobs `ack` after dead-letter.
- `succeeded`, `dead_lettered`, `cancelled`: `ack`.
- missing state or unsupported work: `term`.

## Verification

- `go test ./app/service -run "TestWorkQueueExternalLeaseService|TestAgentJobServiceRecoversExpiredLeases" -count=1 -v`
- `go test ./app/service ./cmd/agent-runtime -count=1`

## Remaining Risk

- The NATS consumer still subscribes only to outbox subjects in live runtime.
- A NATS-level duplicate-delivery smoke is still required before enabling
  `agent_job` external lease subject consumption.
