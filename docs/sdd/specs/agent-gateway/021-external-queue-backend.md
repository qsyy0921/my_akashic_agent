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
execution blocked. A local NATS JetStream smoke has validated
`shadow_publish` + `dual_read_compare` with one outbox work notification,
one matched comparison, and zero mismatches. The sixth slice implements the
first guarded `external_lease` executor for outbox delivery only; agent jobs
remain on Go state-store leasing until their worker idempotency is audited. The
seventh slice adds a local NATS smoke that exercises success ack, retryable
failure delayed nack, terminal failure ack, and unsupported work term without
using real QQ or Telegram adapters. The eighth slice adds an explicit
`agent_job` result-ack scope gate and a NATS-level duplicate terminal delivery
smoke for generic jobs, while still keeping Python responsible for actual model
execution. The ninth slice adds a NATS-level pending/running/succeeded flow
smoke for `agent_job` result-ack and makes that smoke an explicit gate before
expanding live subject consumption.

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
   - Current implementation supports outbox delivery execution only.
   - The consumer subscribes to `akashic.work.outbox.>` and does not execute
     `agent_job` notifications.
   - Retryable dispatch failures move through Go failure/retry state and then
     delayed NATS `nack`; succeeded or terminal failed deliveries NATS `ack`;
     malformed or unsupported work is NATS `term`.
   - `/v1/queue-backend` reports `execution_scope=outbox_delivery_only` when
     the gate is ready. It also reports `agent_job` as a blocked work kind
     because Python worker completion must control queue acknowledgement.
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
$env:AKASHIC_QUEUE_EXTERNAL_LEASE_NACK_DELAY_SECONDS = "30"
$env:AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT = "1049511700=qq_1049511700,2365524513=qq_2365524513"
```

For local Windows smoke runs, Go dependencies can use a domestic module mirror:

```powershell
go env -w GOPROXY="https://goproxy.cn,direct"
go env -w GOSUMDB=off
```

If Docker Hub access is unstable, prefer a Docker registry mirror before
changing runtime queue semantics. On this Windows development machine the
user-level Docker Desktop config lives at
`C:\Users\qsyy0921\.docker\daemon.json`; the mirror fallback should include:

```json
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://docker.1ms.run"
  ]
}
```

The daemon only reports these mirrors after Docker Desktop is restarted. The
mirror itself can be smoke-tested without a daemon restart by pulling a fully
qualified image such as:

```powershell
docker pull docker.m.daocloud.io/library/nats:2-alpine
```

External image and Git operations should prefer the configured local proxy
first, for example `HTTP_PROXY=http://127.0.0.1:7897` and
`HTTPS_PROXY=http://127.0.0.1:7897`. Local runtime calls should keep
`NO_PROXY=127.0.0.1,localhost`.

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
- the execution scope, currently `outbox_delivery_only` after all gates pass;
- allowed work kinds and blocked work kinds;
- required checks and blockers;
- ack, nack, retry, dead-letter, and rollback policies.

`AKASHIC_QUEUE_MODE=external_lease` alone never enables execution. Current
required checks include:

- NATS JetStream provider selected;
- queue DSN configured;
- `AKASHIC_QUEUE_EXTERNAL_LEASE_CUTOVER=true`;
- `AKASHIC_QUEUE_DUAL_READ_SMOKE_PASSED=true`;
- `AKASHIC_QUEUE_STATE_LEASE_WORKERS_DISABLED=true`;
- external lease executor implemented.

The executor implemented in this slice is intentionally scoped to outbox
delivery. It reuses `OutboxService`, `DeliveryDispatchService`, and configured
DeliveryAdapters. It does not introduce a new service process and it does not
execute generic `agent_job` work.

`agent_job` remains intentionally blocked from external lease execution. Unlike
outbox delivery, the side effect is long-running Python work: image generation,
media vision/OCR, group memory extraction, RAG ingest, and RAG eval. A NATS
consumer cannot safely `ack` those messages when Go merely leases the job; it
must wait until the Python worker has completed or failed the job. Moving
`agent_job` to external lease therefore requires a separate result-ack protocol:

- exact `job_id` lease by queue work id, not only `lease-next` by job type;
- lease token or fencing value so stale Python workers cannot complete a newer
  lease;
- worker heartbeat or renewable lease for long model/RAG runs;
- idempotent `complete` / `fail` writeback keyed by job id and lease token;
- queue `ack` only after Go records terminal or retryable job state;
- queue `nack` / delay / `term` mapping aligned with Go domain retry and
  dead-letter policy;
- contract smoke that proves duplicate queue deliveries do not duplicate Python
  side effects.

Current compatibility slice implements the first fencing primitive:

- `AgentJob` stores a generated `lease_token` for each lease attempt.
- `POST /v1/jobs/lease-next` and `POST /v1/jobs/{job_id}/lease` return that
  token in the job view.
- `POST /v1/jobs/{job_id}/running|succeeded|failed` accept optional
  `lease_token`. When present, Go rejects stale or mismatched tokens before
  applying the state transition.
- Python image, knowledge, and RAG-eval workers pass the token from lease to
  running/succeeded/failed writeback.
- Empty-token state updates remain accepted for compatibility with older
  workers; strict token mode is a later cutover gate before external queue
  execution.

The next compatibility slice adds renewable leases for long-running Python
work:

- `POST /v1/jobs/{job_id}/renew` extends the current lease without incrementing
  attempts and records a `renewed` lifecycle event.
- Renew requires a non-empty current `lease_token`; stale tokens are rejected.
- Renew is allowed only for `leased` or `running` jobs whose lease has not
  already expired.
- Python image, knowledge, and RAG-eval workers run a background heartbeat while
  executing a leased job. The heartbeat calls `/renew` before the lease TTL
  expires and stops before result writeback or failure writeback.
- This still keeps `agent_job` on Go state-store leasing; it does not yet allow
  NATS to own generic job acknowledgement.

Strict token mode is an explicit runtime cutover gate:

- `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true` makes `running`, `succeeded`, and
  `failed` transitions reject empty `lease_token` values.
- The default remains compatibility mode so old Python workers can still finish
  state-store leased jobs during rollout.
- `renew` is always strict because a heartbeat without a token cannot safely
  prove ownership.
- Agent-job external queue execution must not be enabled until strict token mode
  is deployed with updated Python workers.

Queue work-id exact leasing is now available for agent jobs:

- `POST /v1/jobs/lease-work` accepts `work_kind`, `work_id`, `aggregate_id`,
  `subject`, `worker_id`, optional `lease_token`, and `ttl_seconds`.
- The handler accepts only `work_kind=agent_job`.
- `work_id` must identify the exact `AgentJob.JobID`; if `aggregate_id` is
  present it must match `work_id`.
- The service delegates to the same domain lease path as
  `POST /v1/jobs/{job_id}/lease`, so attempts, token creation, and lifecycle
  events remain consistent.
- This endpoint is a queue-notification contract boundary; it does not by
  itself run Python work or acknowledge NATS deliveries.

Expired lease recovery is now an explicit Go-owned operation:

- `POST /v1/jobs/recover-expired` scans `leased` and `running` jobs whose
  `lease_expires_at` is older than the supplied timestamp.
- Jobs with remaining attempts are moved back to `pending`, and their lease
  owner, lease token, and expiry are cleared.
- Jobs whose attempt count already reached `max_attempts` are moved to
  `dead_lettered` with an explicit timeout reason.
- Each recovered or dead-lettered job emits a `lease_expired` lifecycle event.
- This operation is intentionally separate from NATS acknowledgement for now.
  Future external consumers should run recovery before duplicate-delivery
  handling, then decide `ack`, delayed `nack`, or `term` from the authoritative
  Go job state.

The same recovery path can run as an optional runtime background job:

- `AKASHIC_AGENT_JOB_RECOVERY_ENABLED=true` starts the runner.
- `AKASHIC_AGENT_JOB_RECOVERY_INTERVAL_SECONDS` controls scan interval, default
  `300`.
- `AKASHIC_AGENT_JOB_RECOVERY_LIMIT` controls per-scan batch size, default `50`,
  max `200`.
- `AKASHIC_AGENT_JOB_RECOVERY_RUN_ON_START` defaults to `true` when the runner is
  enabled.
- The runner is a trigger adapter. It calls the application-level
  `RecoverExpiredLeases` use case and does not duplicate domain rules in the
  runtime process.
- The runner is disabled by default so existing Python state-store workers and
  heartbeat behavior are unchanged unless the operator opts in.

Agent-job result acknowledgement now has a safe Go mapping:

- Go does not execute image generation, RAG ingest, group memory extraction, or
  any other Python-owned model side effect.
- For `agent_job` queue notifications, Go treats the `AgentJob` state store as
  the source of truth and only decides queue disposition.
- `pending` jobs return delayed `nack` with
  `agent_job_pending_for_python_worker`, allowing the Python state-store worker
  to lease and execute the job.
- active `leased` / `running` jobs return delayed `nack` with
  `agent_job_waiting_for_result`; duplicate queue deliveries during execution
  do not create duplicate Python side effects.
- `failed` retryable jobs are moved back to `pending` through the domain retry
  path and then delayed `nack`ed.
- expired active leases are recovered first. Retryable recoveries are delayed
  `nack`ed; exhausted recoveries are `ack`ed because the job has reached
  `dead_lettered`.
- `succeeded`, `dead_lettered`, and `cancelled` jobs are `ack`ed. Replayed
  terminal notifications are therefore idempotent.
- missing job state is `term`ed because queue replay cannot safely recreate a
  missing aggregate.
- Runtime execution scope still remains `outbox_delivery_only`. Moving
  `agent_job` subjects into the live NATS external lease consumer requires a
  NATS-level duplicate-delivery smoke, a NATS-level pending/running/succeeded
  flow smoke, and explicit execution-scope expansion.

Agent-job subject consumption is explicitly gated:

- By default the NATS external lease consumer subscribes only to
  `{subject_prefix}.outbox.>`.
- When the runtime gate allows `agent_job`, the consumer uses
  `{subject_prefix}.>` and the default durable changes from
  `AKASHIC_EXTERNAL_LEASE_OUTBOX` to `AKASHIC_EXTERNAL_LEASE_ALL` to avoid
  reusing an incompatible outbox-only durable.
- `agent_job` is allowed only when base external-lease cutover gates pass and
  all of these flags are set:
  `AKASHIC_QUEUE_EXTERNAL_LEASE_AGENT_JOB_ENABLED=true`,
  `AKASHIC_QUEUE_AGENT_JOB_DUPLICATE_SMOKE_PASSED=true`, and
  `AKASHIC_QUEUE_AGENT_JOB_FLOW_SMOKE_PASSED=true`, and
  `AKASHIC_AGENT_JOB_STRICT_LEASE_TOKEN=true`.
- The expanded scope is reported as
  `outbox_delivery_and_agent_job_result_ack`; this means queue result
  acknowledgement only, not Go execution of model/RAG/memory jobs.

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

The first executor is minimal and reuses existing outbox delivery app services:

```text
NATS outbox notification
        |
        v
ExternalLeaseConsumer
        |
        v
WorkQueueExternalLeaseService
        |
        +--> OutboxService.Lease(work_id)
        +--> DeliveryDispatchService.Dispatch(work_id)
        +--> OutboxService.MarkSucceeded / MarkFailed / Retry
        |
        v
NATS ack / delayed nack / term
```

It should not introduce a new service or package tree unless the deployment
boundary becomes independent.

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
- `external_lease` must consume only outbox subjects until agent job execution
  has separate idempotency and worker-result review.
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
- Local NATS JetStream smoke can create an outbox work notification and observe
  `/v1/queue-backend` reporting `shadow_publish.succeeded_total >= 1`,
  `dual_read_compare.matched_total >= 1`, and
  `dual_read_compare.mismatched_total = 0`.
- `external_lease` mode exposes a blocked cutover gate with required checks and
  policies.
- `external_lease` reports `allow_execution=true` only after explicit cutover,
  dual-read smoke, and legacy state-store lease worker shutdown flags pass.
- `external_lease` outbox execution leases the Go aggregate by work id before
  dispatching any DeliveryAdapter side effect.
- `external_lease` maps delivery success to NATS `ack`, retryable delivery
  failure to Go retry + delayed NATS `nack`, terminal failure to NATS `ack`, and
  malformed/unsupported work to NATS `term`.
- Local `external_lease` smoke verifies those four dispositions with a fake
  DeliveryAdapter and a real NATS JetStream container, without sending real
  platform messages.
- `external_lease` supports an explicitly gated `agent_job` result-ack scope:
  pending/running jobs delayed `nack`, terminal duplicate deliveries `ack`,
  missing state `term`, retryable failures return to pending and delayed
  `nack`, and expired leases recover before disposition.
- Local NATS JetStream smoke verifies duplicate terminal `agent_job`
  notifications both `ack` without invoking Python or platform adapters.
- Local NATS JetStream smoke verifies a single `agent_job` state flow:
  `pending` notification delayed `nack`s for Python, `running` notification
  delayed `nack`s while waiting for the result, and `succeeded` notification
  `ack`s after Python-style result writeback.
- `external_lease` does not execute generic `agent_job` work in this slice.
- Existing outbox/job tests continue to pass.
