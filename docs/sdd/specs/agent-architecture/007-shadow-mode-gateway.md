# SPEC-007: Shadow Mode Gateway

## Status

Draft

## Context

Phase 2 must validate the Go message gateway against real traffic without
changing the working Python QQ, Telegram, CLI, or dashboard runtime. Shadow mode
is a mirror path: it observes normalized Python inbound events, validates the Go
contract, and records loop-guard decisions, but it never becomes the production
send or reply path.

## Ownership

Python owns:

- platform adapters still connected to NapCat, Telegram, CLI, and webhooks;
- live agent execution and current message bus delivery;
- conversion from `InboundMessage` to the shared `MessageEnvelope` JSON shape;
- best-effort shadow delivery that must not block the main inbound queue.

Go owns:

- `/v1/shadow/inbound` validation endpoint;
- provenance classification and loop-guard explanation in shadow mode;
- observed/audit storage for comparison;
- no production outbox behavior in this phase.

Optional persistence:

- `AKASHIC_SHADOW_AUDIT_PATH` enables a Go-side JSONL audit store.
- The JSONL store implements the same audit/query ports as the in-memory
  development store.
- Persistence is for inspection and replay only; it must not become the
  production inbox or outbox.

## Python Shadow Contract

Every mirrored inbound event should include:

- `schema_version`
- `kind = "MessageEnvelope"`
- `event_id`
- top-level `platform`, `account_id`, `conversation_id`, `conversation_type`
- `agent_id`
- `channel`
- `sender`
- `content`
- `attachments`
- timezone-aware `timestamp`
- `source_message_ids`, `source_asset_ids`, and `metadata`

The Python producer must preserve unknown metadata as strings where possible so
the Go API DTO can receive it without rejecting live platform-specific fields.

## Runtime Behavior

- Shadow mode is disabled by default.
- When enabled, `MessageBus.publish_inbound` queues the message for the current
  Python agent first.
- Shadow observers run best-effort after queueing. Observer failures are logged
  and must not fail or delay normal agent processing.
- Local JSONL logging is allowed even when the Go gateway is not running.
- HTTP POST to Go is optional and controlled by configuration.
- Go JSONL audit persistence is optional and must not change Python queueing,
  replying, or observe-only behavior.

## Go Shadow Endpoint

`POST /v1/shadow/inbound` accepts the same inbound message DTO as
`POST /v1/inbound`, but it never publishes `AgentInbound` to Python. It only:

1. validates the normalized message;
2. classifies provenance;
3. applies loop-guard decision logic;
4. records observed/audit entries;
5. returns the decision summary.

## Acceptance

- Python can convert QQ/Telegram/private/group messages into the shared fixture
  shape and preserve metadata.
- Go can ingest the shadow request and report `allow`, `observe_only`, or
  `drop` without forwarding to the agent.
- A shadow endpoint failure does not prevent `MessageBus.consume_inbound`.
- Existing production endpoints and channel behavior remain unchanged.
