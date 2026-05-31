# Inbound Dedupe Runtime State

## Status

Accepted

## Problem

Telegram receiver code had an in-process `MessageDeduper` for duplicate
`(chat_id, message_id)` deliveries. That protects a single Python process, but
it does not survive receiver restarts and keeps another deterministic runtime
state inside the AI worker layer.

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

- `scope`: platform/channel scope such as `telegram:telegram`;
- `message_key`: stable platform message id such as `{chat_id}:{message_id}`;
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

If the runtime call fails, Python logs at debug level and continues with the
local dedupe decision. That preserves current receive behavior when Go is down.

## Boundaries

- Go owns TTL, persistence, and duplicate decision for inbound platform event
  ids.
- Python owns Telegram SDK polling, allowed-user checks, attachment downloads,
  and final `InboundMessage` construction.
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
