# Review: External Queue Backend Design

Spec:

- `docs/sdd/specs/agent-gateway/021-external-queue-backend.md`

Implementation summary:

- Added a read-only queue backend diagnostics view under the Go application
  layer.
- Added `GET /v1/queue-backend` to expose normalized provider, migration mode,
  DSN presence, redacted DSN, active state, consumer model, concurrency, max
  in-flight, and notes.
- Added environment parsing for `AKASHIC_QUEUE_BACKEND`, `AKASHIC_QUEUE_MODE`,
  `AKASHIC_QUEUE_DSN`, `AKASHIC_QUEUE_CONSUMER_CONCURRENCY`, and
  `AKASHIC_QUEUE_MAX_IN_FLIGHT`.
- Kept existing outbox and generic job leasing on Go state stores; no external
  queue adapter is active in this slice.

Tests run:

- `go test ./...` from `services/agent-runtime`.

Findings:

- NATS JetStream is the right first external backend because Akashic is an
  event-heavy Agent Runtime, and subject routing maps cleanly to
  platform/account/job boundaries.
- Bounded Go goroutine worker pools should consume MQ messages concurrently,
  with `consumer_concurrency` and `max_in_flight` exposed in diagnostics.
- Redis Streams is a useful local/simple deployment alternative, but less
  natural for subject-based event routing.
- RabbitMQ is mature but heavier than needed for the current single-runtime
  outbox/job workloads.

Decision:

- Accept the design and expose diagnostics first.
- Do not switch work discovery away from Go state stores until a later
  shadow-publish slice proves external queue behavior.

Follow-ups:

- Implement `shadow_publish` with NATS JetStream after live QQ adapter smoke is
  completed or explicitly deferred.
- Add queue lag/dead-letter diagnostics once a real external adapter exists.
