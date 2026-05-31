# Phase 8.111 Review: External Lease Local Worker Conflict Gate

## Spec

- `docs/sdd/specs/agent-gateway/054-external-lease-local-worker-conflict-gate.md`
- `docs/sdd/TODO.md`

## Implementation Summary

- Added `local_outbox_worker_disabled` to external lease required checks.
- `AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED=true` now blocks
  `/v1/queue-backend.external_lease.allow_execution` even when all other base
  cutover gates pass.
- Kept the existing `startOutboxDeliveryWorker` runtime conflict check as a
  second defense.
- Added command-level test coverage for the conflict gate.

## Tests Run

- `go test ./cmd/agent-runtime -run "TestQueueBackendViewFromEnv.*ExternalLease" -count=1 -v`
- `go test ./...`

## Findings

- No code-review blockers found.
- The change is a read-only gate/preflight safety improvement. It does not
  change local worker behavior, NATS consumer behavior, delivery adapters, or
  Python AI workers.
- The ownership invariant is clearer now: real outbox delivery execution is
  owned by either the local state-store worker or the NATS external lease
  executor, never both.

## Decision

Approved. External lease readiness now fails early when local outbox worker is
still enabled, so cutover diagnostics align with the runtime startup safety
check.

## Follow-ups

- Live preflight before cutover should confirm this blocker disappears only
  after the local outbox worker is disabled.
