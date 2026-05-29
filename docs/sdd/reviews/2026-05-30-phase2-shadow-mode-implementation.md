# Review: Phase 2 Shadow Mode Implementation

Spec:

- `docs/sdd/specs/agent-architecture/005-migration-plan.md`
- `docs/sdd/specs/agent-architecture/007-shadow-mode-gateway.md`
- `docs/sdd/specs/group-message-memory/004-evaluation-and-quality-gates.md`

Implementation summary:

- Add a Python shadow observer that mirrors `InboundMessage` into the shared
  `MessageEnvelope` JSON shape after the normal bus queue receives the message.
- Add optional local JSONL and optional HTTP delivery to Go, both best-effort.
- Add Go `/v1/shadow/inbound` that validates, classifies, and audits without
  publishing agent-inbound work.

Safety boundaries:

- No QQ or Telegram adapter migration.
- No Go production outbox.
- No automatic group replies.
- No replacement of the Python message bus.

Tests run:

- `uv run pytest tests\test_shadow_gateway.py tests\test_sdd_contract_fixtures.py -q`
  passed.
- `uv run pytest tests\test_runtime_smoke.py::test_load_config_keeps_internal_max_iterations_default tests\test_runtime_smoke.py::test_load_config_defaults_memory_window_and_optimizer_interval -q`
  passed with `TMP`/`TEMP` redirected to the workspace because the default
  Windows temp directory was not writable.
- `uv run pytest tests\test_support_modules.py::test_message_bus_covers_flows -q`
  passed with `TMP`/`TEMP` redirected to the workspace.
- `go fmt ./...` passed under `services/message-gateway`.
- `go test ./...` passed under `services/message-gateway`.

Decision:

- Proceed as Phase 2 shadow-only infrastructure.

Follow-ups:

- Add dashboard views for shadow audit and count comparisons.
- Add observe-only session-store mirroring for QQ group messages that bypass the
  inbound bus today.
