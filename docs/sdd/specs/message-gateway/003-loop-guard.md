# SPEC-003: Loop Guard

## Status

Draft

## Context

Bot-to-bot conversations may be useful for delegation and critique, but they
must not run indefinitely.

## Goals

- Allow multi-turn bot-to-bot sessions when explicitly tagged.
- Prevent infinite loops without hard-coding a one-hop limit.
- Produce explainable stop reasons for audit and dashboard display.

## Controls

- `ttl`: root conversation lifetime, default 600 seconds.
- `hop`: processable bot-to-bot hop count, default maximum 6 in the first
  implementation.
- `budget`: remaining processable bot-to-bot messages, reserved for root-session
  protocol.
- `nonce`: each nonce can be consumed once.
- `content_hash`: repeated normalized content under the same root stops the root.
- `progress`: consecutive `no_progress` signals stop the root.
- `root_lock`: one active handler per root.
- `cooldown`: same bot pair cannot immediately restart after stop.
- `send_ledger`: messages sent by Akashic are recorded by content hash and
  short time window, so platform echo is observed but not replied to.

## State Machine

```text
new -> active -> stopped
active -> expired
active -> budget_exhausted
active -> duplicate_content
active -> no_progress
active -> explicit_stop
```

## Invariants

- A consumed nonce is never processed twice.
- Sender equal to the receiving bot account is observe-only.
- Peer-bot plain text is observe-only.
- Hop overflow is dropped.
- Unknown or invalid messages do not consume budget.

## Acceptance Tests

- Replayed nonce is observe-only.
- Untagged peer bot message is observe-only.
- Tagged peer bot message within hop budget is accepted.
- Recent outbound content echo is observe-only.
