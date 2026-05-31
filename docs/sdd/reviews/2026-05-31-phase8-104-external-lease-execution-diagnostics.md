# Phase 8.104 Review: External Lease Execution Diagnostics

Spec:

- `docs/sdd/specs/agent-gateway/047-external-lease-execution-diagnostics.md`

Implementation summary:

- `WorkQueueExternalLeaseService` now records in-memory, bounded diagnostics for
  every external lease execution result.
- `/v1/queue-backend` now includes `external_lease.diagnostics` when the
  external lease gate exists. If no recorder is active, it returns a disabled
  placeholder with an explanatory note.
- Diagnostics include total executions, error total, disposition counters,
  reason counters, work-kind counters, and recent execution samples.
- The change is read-only observability. It does not execute Python AI jobs,
  change AgentJob lease semantics, or send platform messages.

Tests run:

- `go test ./app/service -run "TestWorkQueueExternalLeaseService|TestQueueBackendService"`
- `go test ./cmd/agent-runtime -run TestQueueBackendViewFromEnvReportsExternalLeaseGate`
- `go test ./...`

Findings:

- No blocking findings. The diagnostics are intentionally process-local; durable
  truth remains the Go outbox / AgentJob stores and lifecycle event streams.
- Live NATS smoke is still required to confirm counters move under a real
  external lease consumer.

Decision:

- Accepted.

Follow-ups:

- Use `external_lease.diagnostics` during NATS live smoke before enabling any
  broader external queue cutover.
- Keep AgentJob execution in Python; Go should only acknowledge queue
  notifications after Python result writeback or terminal state recovery.
