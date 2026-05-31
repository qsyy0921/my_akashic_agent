# Inbound Dedupe Runtime State

## Status

Accepted

## Problem

Telegram receiver code had an in-process `MessageDeduper` for duplicate
`(chat_id, message_id)` deliveries. QQ/NapCat receiver code also receives
platform `message_id` for private and group messages, but previously did not
have a restart-safe duplicate event guard. In-process checks protect one Python
process, but they do not survive receiver restarts and keep deterministic
runtime state inside the AI worker layer.

The dedupe decision is not AI behavior. It is infrastructure state: scoped key
tracking, TTL, persistence, and observability. It should live in Go while Python
continues to own platform SDK handling and message-to-agent conversion.

## Decision

`agent-runtime` owns an inbound dedupe bounded context:

```text
POST /v1/inbound-dedupe/check
GET  /v1/inbound-dedupe/records?scope=telegram:telegram&limit=100
```

The check command accepts:

- `scope`: platform/channel/account scope such as `telegram:telegram` or
  `qq:qq_2365524513:2365524513`;
- `message_key`: stable platform message id such as `{chat_id}:{message_id}` or
  `group:{group_id}:{message_id}`;
- `ttl_seconds`: default 24 hours, clamped by the app service;
- `timestamp`: optional RFC3339Nano timestamp for deterministic tests;
- `metadata`: non-secret diagnostic fields such as `message_kind`.

The response returns `duplicate`, first/last seen timestamps, expiry, seen
count, metadata, and `side_effect=runtime_state_only`.

## Storage

The default store is file-backed:

```text
.akashic-workspace/agent-runtime/inbound-dedupe.json
```

Overrides:

```text
AKASHIC_INBOUND_DEDUPE_DSN=/path/to/inbound-dedupe.json
AKASHIC_INBOUND_DEDUPE_PATH=/path/to/inbound-dedupe.json
AKASHIC_INBOUND_DEDUPE_DSN=memory
AKASHIC_RUNTIME_STATE_DIR=memory
```

When persistence is disabled, the shared in-memory runtime store implements the
same port so the HTTP API remains available in ephemeral mode.

## Python Integration

Telegram keeps its local `MessageDeduper` as the first low-latency guard. On a
local miss, it calls Go `check_inbound_dedupe`; if Go reports duplicate, Python
drops the message before typing indicators, downloads, or bus publish.

QQ/NapCat calls Go for events that include a platform `message_id`:

- private messages use `private:{user_id}:{message_id}`;
- normal group messages use `group:{group_id}:{message_id}`;
- observe-only group messages use the same group key and are dropped before
  session write or image download;
- `/stop` messages are checked before interrupt response sends.

Group upload notices remain out of scope for now because their event shape does
not consistently guarantee a stable `message_id`; they should get a separate
file-event idempotency key if needed.

If the runtime call fails, Python logs at debug level and continues with the
local dedupe decision. That preserves current receive behavior when Go is down.

## Boundaries

- Go owns TTL, persistence, and duplicate decision for inbound platform event
  ids.
- Python owns Telegram/QQ SDK polling, allowed-user checks, attachment
  downloads, and final `InboundMessage` construction.
- No platform send, model call, memory extraction, RAG ingest, or dashboard
  mutation is triggered by this endpoint.

## Acceptance

- Repeated checks for the same active `scope + message_key` return
  `duplicate=true` and increment `seen_count`.
- Expired records are removed before the next check and can be accepted as new.
- File-backed state survives `agent-runtime` restart.
- `AKASHIC_RUNTIME_STATE_DIR=memory` keeps the API available with ephemeral
  storage.
- Telegram duplicate runtime responses stop message publication before typing
  side effects.
- QQ private/group duplicate runtime responses stop message publication,
  observe-only session writes, interrupt replies, and attachment downloads for
  stable `message_id` events.
