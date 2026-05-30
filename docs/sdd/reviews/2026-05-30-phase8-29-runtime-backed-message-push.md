# Review: Runtime-backed Message Push

Spec: `docs/sdd/specs/agent-gateway/017-runtime-backed-message-push.md`

Implementation summary:
- Added `AgentRuntimeOutboundEnqueuer` to translate Python `message_push`
  calls into Go `/v1/outbound` payloads.
- Added optional runtime enqueue hook to `MessagePushTool`; configured
  channels enqueue through Go, while unsupported channels keep the direct
  Python sender path.
- Added `execute_direct(...)` to `MessagePushTool` and made the Python outbox
  compatibility worker use it, preventing dispatch-time recursive requeue.
- Added `/v1/outbound` client coverage and bootstrap wiring gated by
  `agent_runtime.enabled`, `agent_runtime.outbox_worker_enabled`, non-empty
  `base_url`, and configured `outbound_channels`.
- Kept the default runtime-backed channel list to Telegram only; QQ/NapCat
  remains direct until the Go adapter boundary is implemented.

Tests run:
- `uv run pytest tests/test_agent_runtime_outbound.py tests/test_agent_gateway_client.py tests/test_agent_gateway_outbox_worker.py tests/test_support_modules.py tests/test_bootstrap_wiring_p2.py -q --basetemp .tmp/pytest-runtime-backed-message-push`

Findings:
- Targeted Python tests passed.
- The outbox worker now explicitly bypasses the runtime-backed `execute(...)`
  path during fallback dispatch.
- Bootstrap account-id mapping is ready for QQ channel names but QQ is not
  enabled in `outbound_channels`.

Decision:
- Accept for Telegram runtime-backed outbound. This moves the normal outbound
  lifecycle into Go without changing QQ direct behavior.

Follow-ups:
- Move QQ/NapCat direct sender fallback behind a Go adapter once the OneBot
  boundary is designed.
- Add contract fixtures covering `/v1/outbound` and outbox delivery dispatch.
