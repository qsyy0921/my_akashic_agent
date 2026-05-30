# SPEC-006: Outbox Delivery And Retry

## Status

Accepted with file-backed recovery and worker leasing for non-production
control-plane migration.

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
- last error message;
- created and updated timestamps.

The implementation supports both the in-memory infrastructure store and an
optional file-backed store configured by `AKASHIC_OUTBOX_DSN` or
`AKASHIC_OUTBOX_PATH`. It remains a control-plane slice: Go owns delivery state,
attempts, retry, lease, and recovery, while production platform sends still
require adapter cutover.

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
  "error_message": "platform timeout"
}
```

## Invariants

- Outbound routing must include platform account id.
- A delivery cannot dispatch after `succeeded` or `dead_lettered`.
- `failed` deliveries can retry until attempts reach max attempts.
- `queued` deliveries are leaseable.
- `dispatching` deliveries are leaseable only after `lease_expires_at`.
- A lease increments attempts and records the worker id.
- `succeeded`, `failed`, and `retry` clear lease fields.
- Exhausted attempts become `dead_lettered`.
- The endpoint is not a production platform sender yet.
- Python compatibility senders may still deliver messages until Phase 4 cutover
  is explicitly reviewed.

## Acceptance Tests

- Domain lifecycle test proves queued -> dispatching -> failed -> retry ->
  dispatching -> dead-letter transitions.
- Application service test proves failed deliveries can be re-enqueued.
- HTTP test proves `/v1/outbound` creates a delivery and `/v1/outbox/*`
  exposes/update states.
- HTTP test proves `/v1/outbox/lease-next` leases the next delivery with owner
  and expiry fields.
- Infrastructure test proves file-backed outbox delivery state and queued retry
  ids survive runtime restarts.
- Infrastructure test proves leased state persists and expired leases become
  leaseable again.
- Go architecture tests continue to enforce DDD dependencies.

## Next Steps

- Add platform delivery adapters behind an outbound port that consumes
  `/v1/outbox/lease-next`.
- Add dashboard outbox/error panel.
- Cut Python channel direct sends only after adapter shadow delivery is proven.
