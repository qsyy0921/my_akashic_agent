# SPEC-020: Outbox Delivery Event Stream

## Status

Implemented initial Go-owned durable event stream.

## Context

The Go runtime already owns outbox delivery state, retries, leases, failure
classification, and platform delivery adapters. Generic agent jobs also have a
durable lifecycle event stream. Outbox deliveries need the same operational
audit trail before QQ/NapCat sends are cut over more broadly, because delivery
bugs are otherwise visible only as the latest state.

## Goals

- Record outbox delivery lifecycle transitions as immutable events.
- Keep the event model separate from `AgentJobEvent` because outbox is a
  different aggregate.
- Persist events through an outbound port and JSONL infrastructure adapter.
- Expose a read-only query endpoint.
- Preserve existing outbox state and delivery behavior.

## Event Types

```text
queued
leased
dispatching
succeeded
failed
retry
```

The event `status` carries the resulting delivery state, so a non-retryable
failure records `event_type=failed` and `status=dead_lettered`.

## Runtime API

```text
GET /v1/outbox-events?limit=50
GET /v1/outbox-events?delivery_id=outbox-http-1
GET /v1/outbox-events?status=dead_lettered
GET /v1/outbox-events?event=failed
```

## Persistence

Environment variables:

```powershell
$env:AKASHIC_OUTBOX_EVENTS_DSN = "E:\agent\akashic\.akashic-workspace\runtime\outbox-events.jsonl"
$env:AKASHIC_OUTBOX_EVENTS_PATH = "E:\agent\akashic\.akashic-workspace\runtime\outbox-events.jsonl"
```

`AKASHIC_OUTBOX_EVENTS_DSN` takes precedence. `memory` keeps events in-process
for tests.

## Boundaries

Go owns:

- event id generation;
- event validation;
- outbox event persistence;
- HTTP read endpoint;
- runtime overview aggregation.

Python owns:

- platform-specific compatibility workers while a channel is not cut over;
- interpreting errors returned by platform sends.

## Acceptance

- Outbound creation records `queued`.
- Lease-next records `leased`.
- Manual dispatching/succeeded/failed/retry actions record their matching event
  type.
- JSONL store reloads and lists events newest-first.
- `/v1/outbox-events` filters by delivery id, status, event type, and limit.
- Runtime overview includes recent outbox event count/details.
- Shared Go/Python contract fixtures include an outbox delivery event stream
  example.
