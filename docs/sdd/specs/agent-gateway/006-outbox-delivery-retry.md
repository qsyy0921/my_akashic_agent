# SPEC-006: Outbox Delivery And Retry

## Status

Accepted with file-backed recovery, worker leasing, and optional Go-owned local
delivery worker for controlled cutover.

## Context

Image generation, Telegram replies, QQ private replies, and future file sends
must not depend on ad hoc Python channel calls forever. The first safe migration
step is to move delivery state into the Go agent gateway while keeping the
actual platform SDK sends on the current Python compatibility path.

This creates a durable boundary contract without cutting over production
delivery prematurely.

## Decision

The Go gateway owns an `OutboxDelivery` aggregate for every outbound request
accepted through `/v1/outbound`.

The aggregate records:

- event id and target channel;
- account id, conversation id, and conversation type;
- content and attachment metadata;
- status: `queued`, `dispatching`, `succeeded`, `failed`, or `dead_lettered`;
- attempts and max attempts;
- lease owner and lease expiry for dispatcher workers;
- structured failure kind for adapter and route diagnostics;
- last error message;
- created and updated timestamps.

The implementation supports both the in-memory infrastructure store and an
optional file-backed store configured by `AKASHIC_OUTBOX_DSN` or
`AKASHIC_OUTBOX_PATH`. Go owns delivery state, attempts, retry, lease, and
recovery. Platform sends happen only through an explicitly enabled worker:
Python compatibility worker for legacy paths, Go local worker for state-store
cutover, or NATS external lease worker for queue cutover.

## HTTP Contract

Create outbound delivery:

```text
POST /v1/outbound
```

List recent deliveries:

```text
GET /v1/outbox?limit=50
```

Read one delivery:

```text
GET /v1/outbox/{event_id}
```

Lease the next queued or expired dispatching delivery:

```text
POST /v1/outbox/lease-next
```

Lease requests include:

```json
{
  "worker_id": "qq-dispatcher",
  "ttl_seconds": 300
}
```

Update state:

```text
POST /v1/outbox/{event_id}/dispatching
POST /v1/outbox/{event_id}/succeeded
POST /v1/outbox/{event_id}/failed
POST /v1/outbox/{event_id}/retry
```

Failure requests include:

```json
{
  "error_kind": "platform_timeout",
  "error_message": "platform timeout"
}
```

Inspect Go-owned outbox metrics:

```text
GET /v1/outbox-metrics?delivery_limit=200&event_limit=200
```

The metrics response summarizes the bounded delivery sample by status and
channel kind, recent lifecycle throughput by event type, current dead-letter
totals, and recent dead-letter samples. It is read-only and does not lease
deliveries, send platform messages, or acknowledge external queue messages.

Run the optional Go local delivery worker:

```powershell
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ENABLED = "true"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_INTERVAL_SECONDS = "2"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_BATCH_SIZE = "1"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_ID = "agent-runtime-outbox-worker"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_LEASE_TTL_SECONDS = "300"
$env:AKASHIC_OUTBOX_DELIVERY_WORKER_RUN_ON_START = "true"
$env:AKASHIC_DELIVERY_CHANNEL_BY_ACCOUNT = "1049511700=qq_1049511700,2365524513=qq_2365524513"
```

This worker is disabled by default. When enabled, it leases from the Go outbox
state store, marks the delivery `dispatching`, calls the Go
`DeliveryDispatchService`, then marks the delivery `succeeded` or `failed`.
It must not run together with NATS external lease execution for outbox delivery,
because both own the same side-effecting dispatch lease.

## Invariants

- Outbound routing must include platform account id.
- A delivery cannot dispatch after `succeeded` or `dead_lettered`.
- retryable `failed` deliveries can retry until attempts reach max attempts.
- non-retryable failure kinds become `dead_lettered` immediately.
- `queued` deliveries are leaseable.
- `dispatching` deliveries are leaseable only after `lease_expires_at`.
- A lease increments attempts and records the worker id.
- `succeeded`, `failed`, and `retry` clear lease fields.
- `failed` records both `error_kind` and `error_message`; unknown or omitted
  kinds are normalized to `unknown`.
- Go local delivery worker is opt-in and uses the same application service
  transitions as HTTP/manual workers.
- Go local delivery worker never dispatches unless a Go `DeliveryAdapter`
  supports the planned channel.
- Go local delivery worker and NATS external lease execution are mutually
  exclusive for outbox delivery.
- `route_error`, `unsupported_media`, and `validation_error` are non-retryable.
- `unknown`, `platform_error`, `platform_timeout`, and `sender_unavailable`
  remain retryable.
- `dispatching`, `succeeded`, and `retry` clear prior failure details.
- Exhausted attempts become `dead_lettered`.
- The endpoint is not a production platform sender yet.
- Python compatibility senders may still deliver messages until adapter cutover
  is explicitly reviewed.
- The opt-in Python compatibility worker may consume `/v1/outbox/lease-next`
  while Go remains the owner of delivery state and retry lifecycle.

## Acceptance Tests

- Domain lifecycle test proves queued -> dispatching -> failed -> retry ->
  dispatching -> dead-letter transitions.
- Application service test proves failed deliveries can be re-enqueued.
- HTTP test proves `/v1/outbound` creates a delivery and `/v1/outbox/*`
  exposes/update states.
- HTTP test proves `/v1/outbox/lease-next` leases the next delivery with owner
  and expiry fields.
- HTTP and app-service tests prove structured failure kind survives state
  updates and is cleared by retry.
- Domain, app-service, and HTTP tests prove non-retryable failures dead-letter
  immediately and cannot be retried.
- Infrastructure test proves file-backed outbox delivery state and queued retry
  ids survive runtime restarts.
- Infrastructure test proves leased state persists and expired leases become
  leaseable again.
- App-service and HTTP tests prove `/v1/outbox-metrics` summarizes delivery
  distribution, event throughput, and dead-letter samples without mutating
  delivery state.
- Job-trigger tests prove the Go local delivery worker leases, marks
  dispatching, dispatches, marks success, maps dispatch errors to structured
  failure kinds, and stays idle when no delivery is leaseable.
- Cmd config tests prove the Go local delivery worker is disabled by default
  and reads bounded env configuration when explicitly enabled.
- Go architecture tests continue to enforce DDD dependencies.

## Next Steps

- Run Go local delivery worker smoke with fake or Telegram adapter before QQ
  NapCat cutover.
- Cut Python channel direct sends only after adapter shadow delivery is proven
  and either Go local worker or NATS external lease owns dispatch.
