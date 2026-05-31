# Phase 8.84 Review: QQ/NapCat Inbound Dedupe Integration

## Scope

- Reused the Go-owned `/v1/inbound-dedupe/check` runtime state from the inbound
  dedupe bounded context.
- Wired Python QQ/NapCat receive paths for private messages, normal group
  messages, observe-only group messages, and `/stop` messages to call Go before
  publishing, writing observe-only session rows, sending interrupt responses, or
  downloading attachments.
- Kept group upload notices out of scope because their event shape does not
  consistently expose a stable platform `message_id`.

## Design Review

- This is aligned with the Go/Python boundary: Go owns deterministic TTL
  duplicate state; Python continues to own NcatBot/NapCat SDK callbacks and
  message conversion.
- The message key includes conversation type and conversation id so private and
  group ids cannot collide even if a provider reuses numeric message ids.
- The scope includes platform, channel alias, and bot account:
  `qq:{channel}:{bot_uin}`. This keeps the two logged-in QQ accounts isolated.
- If Go is unavailable or the event has no `message_id`, Python falls back to
  the previous behavior.
- No platform sends or model calls are introduced by this slice.

## Validation

- `uv run pytest tests\test_channel_clients.py::test_qq_channel_uses_runtime_inbound_dedupe_for_private_message tests\test_channel_clients.py::test_qq_observe_group_uses_runtime_inbound_dedupe tests\test_channel_clients.py::test_qq_group_observe_only_records_without_agent_reply tests\test_channel_clients.py::test_qq_channel_ignores_runtime_send_ledger_echo -q --basetemp .tmp\pytest-qq-inbound-dedupe-target`
- `uv run python -m py_compile infra\channels\qq_channel.py`

## Residual Risk

- QQ/NapCat `message_id` stability is covered by code-level keying and tests,
  but a real reconnect/replay sample should still be inspected before relying
  on this as the only duplicate guard for all QQ event types.
- Group file upload notice idempotency remains a separate TODO because file
  event identity is different from message identity.

## Decision

Accept the QQ/NapCat receive integration as a low-risk extension of the
Go-owned inbound dedupe runtime. It reduces deterministic duplicate-state logic
in Python without changing the observe-only safety boundary.
