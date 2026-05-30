# SPEC-001: Message Envelope

## Status

Draft

## Context

Different channels expose different event shapes. The Agent core should consume
a stable message envelope rather than platform-specific payloads.

## Goals

- Normalize QQ, Telegram, and future platforms into one inbound schema.
- Preserve platform metadata for audit and debugging.
- Make outbound routing explicit by channel and bot account.

## Non-Goals

- Do not perform LLM reasoning in the gateway.
- Do not embed Python session objects in gateway messages.

## Inbound Schema

```json
{
  "event_id": "uuid",
  "platform": "qq",
  "channel": "qq_1049511700",
  "bot_account": "1049511700",
  "chat_type": "private",
  "chat_id": "2365524513",
  "sender_id": "2365524513",
  "content": "hello",
  "media": [],
  "raw": {},
  "provenance": {
    "source": "human",
    "confidence": 1.0,
    "reason": "allow_from"
  },
  "created_at": "2026-05-30T00:00:00+08:00"
}
```

## Outbound Schema

```json
{
  "event_id": "uuid",
  "channel": "qq_1049511700",
  "bot_account": "1049511700",
  "chat_type": "private",
  "chat_id": "2365524513",
  "content": "reply",
  "media": [],
  "reply_to": null,
  "trace": {
    "root_turn_id": "turn-id",
    "purpose": "normal_reply"
  },
  "created_at": "2026-05-30T00:00:00+08:00"
}
```

## Invariants

- `event_id` is globally unique.
- `channel` identifies one concrete bot account route.
- `bot_account` is the account that received or will send the message.
- Raw platform payloads are optional for storage, never required by the Agent.
- Go runtime JSON APIs declare `Content-Type: application/json; charset=utf-8`
  so Chinese QQ/TG content round-trips correctly in Windows PowerShell clients.

## Acceptance Tests

- QQ private message maps to the inbound schema.
- QQ group message maps to the inbound schema.
- Outbound event for `qq_1049511700` is routed only to that account.
- HTTP JSON handlers declare UTF-8 in `Content-Type`.
