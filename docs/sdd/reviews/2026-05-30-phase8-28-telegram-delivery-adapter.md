# Review: Telegram Delivery Adapter

Spec: `docs/sdd/specs/agent-gateway/016-telegram-delivery-adapter.md`

Implementation summary:
- Added a Go Telegram delivery adapter under
  `services/agent-runtime/infrastructure/telegramdelivery`.
- Added `POST /v1/delivery-dispatch/send`, which loads an outbox delivery,
  reuses the Go dispatch planner, and executes supported steps through Go
  adapters.
- Added Python `dispatch_outbox_delivery` client support and changed the
  compatibility outbox worker to prefer Go dispatch for Telegram deliveries.
- Kept QQ/NapCat on the existing Go-plan/Python-send fallback path.

Tests run:
- `go test ./...`
- `uv run pytest tests/test_agent_gateway_client.py tests/test_agent_gateway_outbox_worker.py -q --basetemp .tmp/pytest-telegram-delivery-adapter`
- `git diff --check`

Findings:
- No blocking findings in the implemented slice.
- Telegram live edit, Markdown entity conversion, and inbound polling remain
  Python-owned by design.
- Telegram API 401 is treated as `platform_error`, not fallback
  `sender_unavailable`, so Go adapter registration fallback is not confused
  with real platform/config failures.

Decision:
- Accepted for this migration slice after targeted tests.

Follow-ups:
- Move QQ/NapCat sends behind a Go adapter after Telegram proves stable.
