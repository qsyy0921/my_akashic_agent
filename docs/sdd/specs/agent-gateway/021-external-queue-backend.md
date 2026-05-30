# SPEC-021: External Queue Backend Migration

## Status

In progress. The first implementation slice exposed read-only runtime
diagnostics. The second slice added NATS JetStream `shadow_publish` for outbox
deliveries and generic agent jobs while keeping local state stores
authoritative. The third slice adds shadow publish diagnostics and
state/event-stream reconciliation. The fourth slice adds NATS JetStream
`dual_read_compare`, which consumes queue notifications with a bounded Go
worker pool and compares candidates against Go authoritative state without
executing side effects. The fifth slice adds an `external_lease` cutover gate
that documents and exposes required checks while keeping real external lease
execution blocked.

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
   - Diagnostics are advisory and capped by `sample_limit`; Go state remains the
     source of truth when counts diverge.
   - Publish failure is non-fatal after aggregate state is saved; state-store
     leasing remains the recovery path.

4. `dual_read_compare`
   - Workers still execute leases through Go state store.
   - Queue consumer reads candidate ids and compares them against leaseable
     state before execution.
   - A bounded goroutine worker pool consumes NATS pull messages.
   - Matches and mismatches are recorded as diagnostics, never silently
     executed.
   - Queue messages are acked after compare-only diagnostics are recorded so
     this mode cannot repeatedly execute or re-drive side effects.

5. `external_lease`
   - Queue consumer groups can become the work discovery mechanism only after
     all cutover gates pass.
   - State store still validates idempotency and owns final aggregate state.
   - Failed/expired queue deliveries are reconciled against Go state.
   - Current implementation exposes the gate and policies but blocks execution.
   - Do not add a new service solely for this phase; reuse `agent-runtime`
     application services until deployment or ownership boundaries justify a
     split.

## Runtime Configuration

```powershell
$env:AKASHIC_QUEUE_BACKEND = "local"          # local, nats_jetstream, redis_streams, rabbitmq
$env:AKASHIC_QUEUE_MODE = "local_state_store" # local_state_store, shadow_publish, dual_read_compare, external_lease
$env:AKASHIC_QUEUE_DSN = "nats://127.0.0.1:4222"
$env:AKASHIC_QUEUE_STREAM = "AKASHIC_WORK"
$env:AKASHIC_QUEUE_SUBJECT_PREFIX = "akashic.work"
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

When `mode=shadow_publish`, the response also contains `shadow_publish`:

- publish attempts, successes, and failures;
- per-subject publish counts;
- per-work-kind reconciliation for `outbox_delivery` and `agent_job`;
- sampled Go state count, sampled lifecycle event count, and their deltas
  against successful queue publishes.

When `mode=dual_read_compare`, the response also contains
`dual_read_compare`:

- total compared queue candidates;
- match and mismatch counts;
- mismatch reasons such as `missing_state`, `not_leaseable`, or
  `unsupported_work_kind`;
- recent candidate comparison samples.

When `mode=external_lease`, the response also contains `external_lease`:

- whether explicit cutover was requested;
- whether execution is allowed;
- required checks and blockers;
- ack, nack, retry, dead-letter, and rollback policies.

`AKASHIC_QUEUE_MODE=external_lease` alone never enables execution. Current
required checks include:

- NATS JetStream provider selected;
- queue DSN configured;
- `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER=true`;
- `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED=true`;
- external lease executor implemented.

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

NATS subjects:

```text
akashic.work.outbox.{channel_kind}.{account_id}
akashic.work.agent_job.{job_type}
```

Notification payloads include:

- `schema_version`
- `work_kind`
- `work_id`
- `aggregate_id`
- route and status hints
- source event ids and source asset ids
- metadata
- timestamp

The payload is intentionally a work notification. Consumers must read/lease the
authoritative aggregate through Go APIs before executing side effects.

`dual_read_compare` adds an inbound compare-only application port:

```go
type WorkQueueCandidateComparer interface {
    CompareWorkQueueCandidate(ctx context.Context, cmd CompareWorkQueueCandidateCommand) (QueueCandidateComparisonView, error)
}
```

This port validates candidate ids against Go state stores and updates
diagnostics. It does not acquire leases or execute side effects.

Add consumer/ack ports only when implementing `external_lease`:

```go
type WorkQueueConsumer interface {
    ClaimOutboxDelivery(ctx context.Context, workerID string, ttl time.Duration) (WorkItem, bool, error)
    Ack(ctx context.Context, workID string) error
    Nack(ctx context.Context, workID string, reason string) error
}
```

The first executor should be minimal and should reuse existing outbox/job app
services. It should not introduce a new service or package tree unless the
deployment boundary becomes independent.

## Invariants

- Creating work must commit Go aggregate state before publishing any queue
  notification.
- Queue messages carry only ids, route hints, and trace metadata; they do not
  carry authoritative aggregate state.
- Shadow publish failures do not fail `/v1/outbound` or `/v1/jobs` after state
  has been committed.
- Worker execution must still transition Go lifecycle state before platform or
  Python side effects.
- Duplicate queue messages are harmless because aggregate ids are idempotent.
- `dual_read_compare` must not call platform adapters, Python workers, or
  delivery dispatch.
- `external_lease` must remain blocked unless its gate reports
  `allow_execution=true`.
- `external_lease` ack/nack policy must be visible before execution is enabled.
- Switching provider must not change HTTP contracts for `/v1/outbox`,
  `/v1/jobs`, `/v1/job-events`, or `/v1/outbox-events`.

## Acceptance

- `/v1/queue-backend` returns local defaults without requiring NATS/Redis/RabbitMQ.
- `AKASHIC_QUEUE_BACKEND=nats` normalizes to `nats_jetstream`.
- Consumer concurrency and max in-flight settings are validated and exposed.
- DSNs are redacted in runtime output.
- Non-local queue configuration does not activate external leasing yet.
- `shadow_publish` can publish NATS JetStream notifications for new outbox
  deliveries and agent jobs.
- `shadow_publish` exposes publish success/failure diagnostics by subject.
- `shadow_publish` reconciles sampled queue publish counts against Go state
  stores and lifecycle event streams.
- `dual_read_compare` starts a bounded NATS pull consumer when configured.
- `dual_read_compare` records match/mismatch diagnostics without acquiring
  external leases or executing side effects.
- `external_lease` mode exposes a blocked cutover gate with required checks and
  policies.
- `external_lease` remains unable to execute until an executor is implemented
  and explicit cutover flags pass.
- Existing outbox/job tests continue to pass.
