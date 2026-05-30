# SPEC-022: Send Ledger Metrics

## Status

Implemented read-only Go-owned metrics endpoint.

## Context

The Go runtime already owns the send ledger used by recent-send and private
echo checks. That ledger is the deterministic state boundary for bot-to-bot loop
protection, especially when two QQ bot accounts can send messages to each
other. Operators need a Go-owned summary of ledger coverage and repeated content
hashes before enabling broader QQ adapter cutover.

## Goals

- Keep recent-send metric semantics inside Go, where the send ledger repository
  is authoritative.
- Summarize bounded ledger samples by bot account, conversation, and content
  hash.
- Surface repeated content hashes per bot/conversation so echo-loop risk is
  visible before live-send smoke.
- Keep the endpoint read-only; it must not record sends, query external
  platforms, or mutate loop-guard state.

## HTTP Contract

```text
GET /v1/send-ledger/metrics?limit=200
GET /v1/send-ledger/metrics?from_bot_id=1049511700&conversation_id=2365524513&limit=200
```

The response includes:

- `sampled_records`;
- `unique_bots`;
- `unique_conversations`;
- `unique_content_hashes`;
- `repeated_content_hashes`;
- `records_by_bot`;
- `records_by_conversation`;
- `repeated_hashes`;
- `recent`.

## Boundaries

Go owns:

- send ledger record storage;
- recent-send and private echo checks;
- bounded metrics over that ledger.

Python owns:

- platform-specific compatibility sends until cutover;
- model/tool behavior and message generation.

The metrics endpoint must not replace `/v1/send-ledger/recent` or
`/v1/send-ledger/private-echo`; it only summarizes ledger state for operators.

## Acceptance

- App-service tests cover bot/conversation distribution and repeated hash
  metrics.
- HTTP tests cover `GET /v1/send-ledger/metrics`.
- Runtime overview dashboard reads the Go endpoint and renders a read-only card.
- Existing recent-send and private echo tests remain green.
