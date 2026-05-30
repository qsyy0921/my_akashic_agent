# Review: Phase 8.73 Delivery Adapter Health

## Changes

- Added Go-owned `GET /v1/delivery-adapters/health`.
- OneBot/NapCat probes call read-only `get_login_info` over HTTP or WebSocket
  action transport.
- Telegram probes call read-only `getMe`.
- Added `AgentGatewayClient.check_delivery_adapter_health()` for Python
  compatibility code and diagnostics.
- Updated README, SPEC-018, and the Chinese TODO.

## Review Notes

- This endpoint is explicit and read-only. It does not create outbox records,
  call `DispatchDeliveryStep`, send platform messages, or mutate runtime state.
- Health is a connectivity/authentication preflight. It does not prove that a
  target private chat, group chat, image upload, or file upload will succeed.
- The endpoint may touch live OneBot/Telegram APIs when called. Runtime overview
  does not call it automatically.

## Verification

- `go test ./app/service ./trigger/http ./cmd/agent-runtime ./infrastructure/onebotdelivery ./infrastructure/telegramdelivery`
- `uv run pytest tests\test_agent_gateway_client.py -q --basetemp .tmp\pytest-delivery-adapter-health-client`
- `go test ./...`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_agent_gateway_outbox_worker.py tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-delivery-adapter-health-final-regression`
- `git diff --check`

## Decision

Accept as the safe pre-send readiness gate before live QQ/NapCat adapter smoke.
