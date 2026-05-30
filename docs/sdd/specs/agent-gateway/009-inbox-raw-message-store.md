# SPEC-009: Go Inbox Raw Message Store

## Status

Implemented initial observe-only slice.

## Context

QQ group observation needs a stable raw message source before Python memory,
RAG, and guide extraction can be trusted. Python currently stores group messages
in session history for compatibility, while Go already receives shadow inbound
events, loop decisions, and media asset registrations.

The next migration step is for Go to own the append-only raw inbox for observed
messages. Python remains responsible for expensive AI work, but it should not be
the only durable source of group message history.

## Goals

- Persist normalized inbound/shadow messages as immutable `InboxEvent` records.
- Keep account id, platform, conversation id, sender id, attachments,
  provenance, loop decision, and observe-only state together.
- Make duplicate event ids idempotent.
- Expose a typed read API for dashboards, replay, memory extraction, and future
  Go-owned queues.
- Keep current QQ observe-only behavior unchanged: storing raw messages must not
  create group-visible replies.

## Non-Goals

- Do not move LLM/VLM interpretation, group-memory extraction, or RAG ranking to
  Go.
- Do not replace the Python session store in this slice.
- Do not cut over QQ/NapCat platform adapters to Go yet.

## Domain Model

`InboxEvent` is a Go domain aggregate:

```text
InboxEvent
├── Envelope: MessageEnvelope
├── Decision: LoopDecision
└── ReceivedAt: time
```

Invariants:

- `Envelope.Validate()` must pass.
- `Decision.Action` is required.
- `ReceivedAt` is required.
- `observe_only` is derived from metadata or an observe-only loop decision.
- Duplicate `event_id` saves are no-ops to keep the store idempotent.

## Ports

Inbound app port:

```text
InboxEventViewer
├── GetInboxEvent(event_id)
└── ListInboxEvents(filter)
```

Outbound app port:

```text
InboxEventRepository
├── SaveInboxEvent(event)
├── FindInboxEvent(event_id)
└── ListInboxEvents(filter)
```

Filters:

- `channel_kind` / `platform`
- `account_id`
- `conversation_id`
- `conversation_type`
- `sender_id`
- `decision_action`
- `observe_only`
- `after_seq`
- `order=asc|desc`
- `limit`

## HTTP API

```text
GET /v1/inbox?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&limit=50
GET /v1/inbox?channel_kind=qq&conversation_id=27234224&conversation_type=group&observe_only=true&after_seq=120&order=asc&limit=80
GET /v1/inbox/{event_id}
```

## Runtime Configuration

```powershell
$env:AKASHIC_INBOX_DSN = "E:\agent\akashic\.akashic-workspace\runtime\inbox.json"
```

`AKASHIC_INBOX_PATH` is accepted as a shorthand. The special value `memory`
keeps the development in-memory store.

## Acceptance

- Shadow and normal ingest write an `InboxEvent` before publishing agent inbound
  work.
- `/v1/inbox` lists raw observed group messages with attachment metadata.
- `/v1/inbox` supports cursor-safe replay for session-backed messages by
  filtering on `metadata.seq` via `after_seq` and returning oldest-first batches
  with `order=asc`.
- `/v1/inbox/{event_id}` returns one raw event.
- File-backed inbox survives `agent-runtime` restart.
- Go unit tests cover domain validation, application view mapping, HTTP API,
  and file-backed persistence.

## Follow-Ups

- Move more historical backfill and replay leases into Go-owned projections once
  group-memory extraction no longer needs the compatibility session store.
- Add a durable stream/lease projection for group extraction workers.
- Merge inbox source ids with media asset ids in RAG citations.
