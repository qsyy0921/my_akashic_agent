# Phase 8.107 Review: Outbox Account Rate Limit

## Spec

- `docs/sdd/specs/agent-gateway/050-outbox-account-rate-limit.md`
- `docs/sdd/TODO.md`

## Implementation Summary

- Added `ChannelRef.AccountKey()` as the stable `channel_kind:account_id`
  account key helper.
- Extended `LeaseNextOutboxCommand` and the outbox repository port with
  `BlockedAccountKeys`, so the application service can skip throttled accounts
  before taking a lease.
- Updated in-memory and file-backed outbox stores to skip blocked accounts while
  still leasing other eligible accounts.
- Added optional Go local outbox delivery worker rate limiting:
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MIN_INTERVAL_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_WINDOW_SECONDS`
  - `AKASHIC_OUTBOX_DELIVERY_ACCOUNT_MAX_PER_WINDOW`
- Exposed configured throttling values in runtime worker diagnostics.
- Documented the worker env settings in `services/agent-runtime/README.md`.

## Tests Run

- `go test ./app/service ./trigger/job ./cmd/agent-runtime ./infrastructure/outboxstore -run "TestOutbox.*Rate|TestOutbox.*Blocked|TestOutboxServiceLeaseNextSkipsBlockedAccountKeys|TestOutboxDeliveryWorkerConfigFromEnv|TestRuntimeWorkerDiagnosticsFromEnvIncludesConfiguredWorkers|TestOutboxStore" -count=1 -v`
- `go test ./...`

## Findings

- No code-review blockers found.
- The rate limiter is intentionally local in-memory worker state. It protects
  the side-effecting local worker without creating a new durable policy store.
- The key safety invariant holds: throttled deliveries are skipped before lease,
  so they remain queued and attempts do not increase.
- NATS `external_lease` outbox execution is not rate-limited in this slice;
  that remains a separate cutover-sensitive follow-up.

## Decision

Approved as a pragmatic Go runtime safety guard for the local outbox delivery
worker. It moves a concrete piece of delivery infrastructure into Go while
keeping Python limited to AI/runtime worker execution.

## Follow-ups

- Live smoke with two QQ accounts before enabling worker on real outbound
  channels.
- Design persisted/distributed throttling only if multiple side-effecting
  delivery executors are enabled.
- Add external lease throttling only after the NATS cutover gates and smoke
  tests are explicit.
