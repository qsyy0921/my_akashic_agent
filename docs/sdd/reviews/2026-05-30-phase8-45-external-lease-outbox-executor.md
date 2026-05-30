# Review: external lease outbox executor

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added a Go application service for `external_lease` outbox execution.
- Added a NATS JetStream external lease consumer that subscribes only to outbox
  subjects and uses a bounded goroutine worker pool.
- The executor leases the Go outbox aggregate by queue `work_id`, dispatches
  through configured Go DeliveryAdapters, then updates Go outbox lifecycle
  state before returning queue disposition.
- Queue disposition policy:
  - success: Go `succeeded` then NATS `ack`;
  - retryable dispatch failure: Go `failed` then Go `retry` then delayed NATS `nack`;
  - terminal failure: Go `dead_lettered` then NATS `ack`;
  - malformed or unsupported work: NATS `term`;
  - already terminal Go state: NATS `ack`.
- No new service process was added. The slice stays inside
  `services/agent-runtime` and reuses the existing DDD/hexagonal boundaries.

Tests run:
- `go test ./...` from `services/agent-runtime`.

Findings:
- No correctness blocker found in unit coverage.
- The first executor is intentionally outbox-only. Generic `agent_job` work is
  still leased through Go state stores because Python worker idempotency and
  result replay need a separate review before MQ ownership moves.
- Real `external_lease` runtime execution remains guarded by
  `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER=true` and
  `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED=true`.
- A separate `AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED=true` gate is required
  so the NATS executor cannot run while legacy state-store lease workers are
  still active.

Decision:
- Proceed with the outbox-only executor as the smallest useful migration step.
- Keep QQ/NapCat live sending disabled until the user explicitly authorizes a
  real-send smoke.

Follow-ups:
- Run an external lease smoke with a fake or controlled adapter so the NATS
  ack/nack/term policy is verified without sending real QQ/Telegram messages.
- Decide separately whether `agent_job` should move to NATS external lease after
  Python worker idempotency is proven.
