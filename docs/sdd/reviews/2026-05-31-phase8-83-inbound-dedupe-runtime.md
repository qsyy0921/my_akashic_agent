# Phase 8.83 Review: Go-Owned Inbound Dedupe Runtime

## Scope

- Added `InboundDedupeRecord` domain model, app command/query ports, service,
  HTTP routes, file-backed store, and in-memory runtime-store implementation.
- Added `POST /v1/inbound-dedupe/check` and
  `GET /v1/inbound-dedupe/records`.
- Added Python `AgentGatewayClient.check_inbound_dedupe`.
- Wired Telegram text/photo receive path to use Go runtime dedupe after the
  existing local process dedupe guard.
- Updated runtime config diagnostics to expose
  `AKASHIC_INBOUND_DEDUPE_DSN/PATH`.

## Design Review

- The slice is DDD-compatible: duplicate detection policy is in app/domain
  layers, persistence is behind an outbound port, and HTTP is only an inbound
  adapter.
- The endpoint is safe for observe-only operation: it mutates only runtime
  dedupe state and does not send messages, run tools, or invoke models.
- Python keeps a local fallback so Telegram receive does not become dependent
  on Go availability.
- Runtime state defaults are consistent: file-backed by default, explicit
  `memory` opt-out remains supported.

## Validation

- `go test ./app/service ./infrastructure/inbounddedupestore ./trigger/http`
- `go test ./...`
- `go vet ./...`
- `go build -o .tmp\bin\agent-runtime.exe .\cmd\agent-runtime`
- `uv run pytest tests\test_agent_gateway_client.py::test_agent_gateway_client_checks_inbound_dedupe tests\test_channel_clients.py::test_telegram_channel_uses_runtime_inbound_dedupe -q`
- `uv run pytest tests\test_agent_gateway_client.py tests\test_channel_clients.py -q --basetemp .tmp\pytest-inbound-dedupe-channel-full`
- `uv run python -m py_compile infra\channels\telegram_channel.py integrations\agent_gateway.py`
- Isolated live smoke on `:18782`: first `/v1/inbound-dedupe/check` returned
  `duplicate=false`; after restarting `agent-runtime` with the same
  `AKASHIC_RUNTIME_STATE_DIR`, the same key returned `duplicate=true` and
  `seen_count=2`.
- Local `:8780` runtime restarted with the new binary; `/healthz` returned OK
  and `/v1/inbound-dedupe/check` returned `side_effect=runtime_state_only`.

## Residual Risk

- Only Telegram is wired in this slice. QQ/NapCat duplicate event ids should be
  evaluated separately because OneBot implementations differ in message id
  stability and resend semantics.
- The JSON file can grow under very high traffic. TTL cleanup happens during
  checks; periodic compaction can be added later if needed.

## Decision

Accept this slice as a low-risk migration of deterministic inbound duplicate
state from Python process memory to Go `agent-runtime`.
