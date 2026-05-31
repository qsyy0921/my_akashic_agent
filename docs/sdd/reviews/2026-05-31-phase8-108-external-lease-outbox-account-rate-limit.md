# Phase 8.108 Review: External Lease Outbox Account Rate Limit

## Spec

- `docs/sdd/specs/agent-gateway/051-external-lease-outbox-account-rate-limit.md`
- `docs/sdd/specs/agent-gateway/050-outbox-account-rate-limit.md`
- `docs/sdd/TODO.md`

## Implementation Summary

- Extracted reusable `OutboxAccountRateLimiter` into Go domain service, covering
  account min interval, rolling window max, pruning, and sorted diagnostics.
- Injected the shared limiter into the local outbox delivery worker from the
  command composition root, keeping `trigger/job` dependent only on a small
  interface instead of importing domain service directly.
- Added the same limiter to `WorkQueueExternalLeaseService` for `outbox`
  work execution. If an account is currently blocked, external lease execution
  returns `nack` with `delivery_rate_limited` before taking an outbox lease or
  calling the delivery adapter.
- Exposed the configured account-limit attributes on both local outbox worker
  diagnostics and NATS `external_lease` runtime worker diagnostics.
- Updated iteration governance prompt to make Python AI runtime responsibilities
  explicit and to require each iteration to clear the current TODO set.

## Tests Run

- `go test ./domain/service ./trigger/job ./app/service ./cmd/agent-runtime -run "TestOutboxAccountRateLimiter|TestOutboxDeliveryWorkerRateLimit|TestWorkQueueExternalLeaseServiceRateLimitsOutbox|TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers|TestOutboxDeliveryWorkerConfigFromEnv" -count=1 -v`
- `go test ./...`

## Findings

- No code-review blockers found.
- The first full-test run exposed a useful layering issue: `trigger/job`
  must not import domain services directly. The final design injects an
  interface from the composition root, preserving the existing DDD/hexagonal
  dependency direction.
- The external lease limiter is intentionally process-local. This matches the
  current local worker policy and avoids introducing distributed coordination
  before real cutover evidence requires it.
- Safety invariant holds: rate-limited external outbox work does not acquire an
  outbox lease, does not increment attempts, and does not dispatch to QQ or
  Telegram adapters.

## Decision

Approved. This closes the immediate bypass risk where future NATS
`external_lease` outbox execution could skip the account throttling already
implemented for the local Go worker, while leaving Python AI/model execution
untouched.

## Follow-ups

- Live smoke NATS `external_lease` with real account throttling before any
  production cutover.
- Add persisted or distributed rate-limit state only if multiple side-effecting
  delivery executors need to run concurrently.
- Keep platform-specific risk policies configurable in Go runtime, not in
  Python prompts.
