# SPEC-021: External Queue Backend Migration

## Status

Design accepted; first implementation slice exposes read-only runtime
diagnostics and keeps local state stores authoritative.

## Context

`agent-runtime` now owns deterministic lifecycle control for:

- outbound delivery state and retries;
- generic agent jobs for image generation, group memory, RAG ingest, and RAG
  eval;
- durable lifecycle event streams.

The current queue behavior is embedded in Go-owned state stores: in-memory for
development or JSON/JSONL files for restart recovery. This is acceptable for a
single local runtime, but it is not enough for multi-worker scheduling,
backpressure, consumer isolation, or production-style operations.

## Decision

Adopt NATS JetStream as the first external queue backend, while keeping the
application ports provider-neutral.

NATS JetStream is the first target because it best matches this Agent Runtime:

- subject-based routing maps naturally to platform/account/job types;
- pull consumers and durable streams fit outbox/jobs without blocking HTTP;
- Go client support is mature and lightweight;
- queue semantics can evolve into the wider event backbone for observed
  messages, media tasks, memory extraction, RAG ingest, proactive scheduling,
  and audit events;
- multiple worker goroutines can consume the same durable consumer with bounded
  concurrency and max in-flight limits.

Redis Streams remains a simple local deployment alternative. RabbitMQ remains a
valid future adapter when exchange routing and enterprise broker operations
become more important than a lightweight event backbone.

## Provider Comparison

| Backend | Strength | Cost / Risk | Fit |
| --- | --- | --- | --- |
| NATS JetStream | Go-native event backbone, subjects fit platform/account/job routing, pull consumers support bounded concurrency | Requires stream/consumer naming discipline | First implementation |
| Redis Streams | Simple local ops, consumer groups, inspectable pending entries | Less natural for subject-based event routing | Local/simple deployment alternative |
| RabbitMQ | Mature acknowledgements, routing exchanges, dead letters | Heavier broker model for current single-node runtime | Later if routing grows complex |

## Target Boundary

The state store remains the source of truth through the migration. External
queue messages are work signals, not canonical aggregate state.

```text
HTTP / Python worker
        |
        v
Go App Service
        |
        +--> State store: OutboxDelivery / AgentJob aggregate state
        |
        +--> Event stream: immutable lifecycle events
        |
        +--> Queue backend: durable work notification and consumer groups
```

This keeps domain recovery deterministic: if NATS/Redis/RabbitMQ loses a
message or is disabled, Go can still discover leaseable work from state.

## Migration Phases

1. `local_only`
   - Current behavior. Outbox and jobs lease directly from Go stores.
   - `/v1/queue-backend` reports provider, mode, and active state.

2. `shadow_ready`
   - Runtime can be configured with an external provider and DSN.
   - No external publish/lease happens yet.
   - Operators can validate configuration without risking duplicated work.

3. `shadow_publish`
   - On outbox/job creation, Go writes aggregate state first, then publishes a
     queue notification.
   - Workers still lease from state store.
   - Diagnostics compare queue notification counts against state/event counts.

4. `dual_read_compare`
   - Workers still execute leases through Go state store.
   - Queue consumer reads candidate ids and compares them against leaseable
     state before execution.
   - Mismatches are recorded as diagnostics, never silently executed.

5. `external_lease`
   - Queue consumer groups become the work discovery mechanism.
   - State store still validates idempotency and owns final aggregate state.
   - Failed/expired queue deliveries are reconciled against Go state.

## Runtime Configuration

```powershell
$env:AKASHIC_QUEUE_BACKEND = "local"          # local, nats_jetstream, redis_streams, rabbitmq
$env:AKASHIC_QUEUE_MODE = "local_state_store" # local_state_store, shadow_publish, dual_read_compare, external_lease
$env:AKASHIC_QUEUE_DSN = "nats://127.0.0.1:4222"
$env:AKASHIC_QUEUE_CONSUMER_CONCURRENCY = "8"
$env:AKASHIC_QUEUE_MAX_IN_FLIGHT = "64"
```

`AKASHIC_QUEUE_BACKEND` defaults to `local`. Non-local providers default to
`shadow_publish` for diagnostics, but the first implementation keeps
`external_queue_active=false` until a concrete adapter is added.

## Runtime API

```text
GET /v1/queue-backend
```

The endpoint returns:

- normalized provider and mode;
- migration phase;
- whether an external queue DSN is configured;
- whether an external adapter is active;
- whether state stores are authoritative;
- consumer model, concurrency, and max in-flight settings;
- source of outbox and generic job work discovery;
- redacted DSN and operational notes.

## Concurrent Consumption

Go should consume MQ work with a bounded goroutine worker pool:

```text
NATS pull consumer
        |
        v
bounded fetch loop
        |
        v
goroutine worker pool, size = AKASHIC_QUEUE_CONSUMER_CONCURRENCY
        |
        v
Go lifecycle transition -> side effect -> ack/nack
```

Rules:

- `AKASHIC_QUEUE_CONSUMER_CONCURRENCY` controls active worker goroutines per
  runtime process.
- `AKASHIC_QUEUE_MAX_IN_FLIGHT` caps fetched but unfinished messages.
- `max_in_flight >= consumer_concurrency` in real adapters.
- A worker must acquire/validate the Go aggregate lease before executing a side
  effect.
- Ack only after Go state reaches `succeeded`, `failed`, `dead_lettered`, or a
  deliberate retry state.
- Nack or terminate queue work when Go rejects the aggregate lease.

## Application Ports

Future queue adapters should not replace aggregate repositories. Add a
provider-neutral queue port only when implementing `shadow_publish`:

```go
type WorkQueuePublisher interface {
    PublishOutboxDelivery(ctx context.Context, delivery model.OutboxDelivery) error
    PublishAgentJob(ctx context.Context, job model.AgentJob) error
}
```

Add consumer/ack ports only when implementing `dual_read_compare` or
`external_lease`:

```go
type WorkQueueConsumer interface {
    ClaimOutboxDelivery(ctx context.Context, workerID string, ttl time.Duration) (WorkItem, bool, error)
    Ack(ctx context.Context, workID string) error
    Nack(ctx context.Context, workID string, reason string) error
}
```

## Invariants

- Creating work must commit Go aggregate state before publishing any queue
  notification.
- Queue messages carry only ids, route hints, and trace metadata; they do not
  carry authoritative aggregate state.
- Worker execution must still transition Go lifecycle state before platform or
  Python side effects.
- Duplicate queue messages are harmless because aggregate ids are idempotent.
- Switching provider must not change HTTP contracts for `/v1/outbox`,
  `/v1/jobs`, `/v1/job-events`, or `/v1/outbox-events`.

## Acceptance

- `/v1/queue-backend` returns local defaults without requiring NATS/Redis/RabbitMQ.
- `AKASHIC_QUEUE_BACKEND=nats` normalizes to `nats_jetstream`.
- Consumer concurrency and max in-flight settings are validated and exposed.
- DSNs are redacted in runtime output.
- Non-local queue configuration does not activate external leasing yet.
- Existing outbox/job tests continue to pass.
- TODO and review records document that actual adapter implementation remains a
  separate migration slice.
