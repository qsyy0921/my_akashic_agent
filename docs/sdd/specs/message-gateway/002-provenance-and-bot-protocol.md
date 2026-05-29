# SPEC-002: Provenance And Bot Protocol

## Status

Draft

## Context

When multiple bot accounts are online, normal peer-bot messages must not trigger
uncontrolled replies. Only explicit bot-to-bot protocol messages may be
processed.

## Goals

- Classify inbound messages as human, self echo, peer bot, or system.
- Require a visible `[[akashic:bot ...]]` tag for bot-to-bot processing.
- Treat peer-bot messages without this explicit tag as observe-only.

## Protocol Tag

```text
[[akashic:bot from=1049511700 nonce=abc123 hop=2]]
body
```

## Signature Roadmap

The first implemented version uses a visible protocol tag plus nonce and hop
limits. HMAC signatures are reserved for the production NATS/NapCat adapter
phase, where a shared secret can be deployed safely.

## Provenance Sources

- `human`: sender is in the human allowlist.
- `self_echo`: sender equals current bot account or matches recent outbound
  ledger.
- `peer_bot`: sender is a known bot account. `has_protocol_tag=true` means the
  message carries parseable protocol metadata.
- `system`: platform/system-generated messages.

## Invariants

- Peer-bot messages without protocol tag never enter Agent reasoning.
- Sender equal to the receiving bot account is self echo.
- Replayed nonce is observe-only.
- Hop budget overflow is dropped.

## Acceptance Tests

- Human message is processable.
- Peer bot plain text is observe-only.
- Peer bot tagged message within hop budget is processable.
- Replayed nonce is observe-only.
