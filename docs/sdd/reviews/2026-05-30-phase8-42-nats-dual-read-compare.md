# Review: NATS dual read compare

Spec:
- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:
- Added a compare-only `WorkQueueCandidateComparer` application port and
  `WorkQueueCompareService` use case.
- Added NATS JetStream `CompareConsumer`, which pull-consumes work
  notifications with bounded goroutine workers and calls the compare-only app
  service.
- In `dual_read_compare`, queue notifications are compared against Go
  authoritative outbox/job state. The service records matches, mismatch
  reasons, work-kind totals, and recent samples for `/v1/queue-backend`.
- `dual_read_compare` continues to publish work notifications, but it does not
  acquire external leases, call delivery adapters, send QQ/Telegram messages, or
  execute Python workers.

Tests run:
- `go test ./...` from `services/agent-runtime`.

Findings:
- No correctness blockers found in this slice.
- The NATS consumer is implemented but live broker behavior is not smoke-tested
  in this slice because no local NATS JetStream server was started.
- The compare diagnostics are in-memory for the runtime process. This is
  acceptable for migration visibility, but durable compare logs may be useful
  before production cutover.

Decision:
- Proceed with `dual_read_compare` as the next MQ migration step. It exercises
  external queue consumption without changing the source of truth or executing
  side effects.

Follow-ups:
- Run a local NATS JetStream smoke test and verify `/v1/queue-backend`
  `shadow_publish` and `dual_read_compare` counts move as expected.
- Design `external_lease` ack/nack, lease acquisition, retry, and dead-letter
  gates before moving real work discovery away from Go state-store leasing.
