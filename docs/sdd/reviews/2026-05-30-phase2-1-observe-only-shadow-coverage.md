# Review: Phase 2.1 Observe-Only Shadow Coverage

Spec:

- `docs/sdd/specs/agent-architecture/007-shadow-mode-gateway.md`
- `docs/sdd/specs/group-message-memory/004-evaluation-and-quality-gates.md`

Implementation summary:

- Mirror observe-only session messages through a `SessionManager` message
  observer instead of adding more behavior to QQ channel handlers.
- Convert persisted observe-only group messages into the same shadow
  `MessageEnvelope` path used by normal bus events.
- Add Go `/v1/shadow/observed` to inspect recent shadow decisions without
  exposing production outbox behavior.

Safety boundaries:

- QQ observe-only groups still do not publish agent inbound events.
- Go shadow query is read-only.
- Python shadow mirror failures are logged and do not block session persistence.

Tests run:

- `uv run pytest tests\test_shadow_gateway.py tests\test_sdd_contract_fixtures.py -q`
  passed.
- `uv run pytest tests\test_runtime_smoke.py::test_load_config_keeps_internal_max_iterations_default tests\test_runtime_smoke.py::test_load_config_defaults_memory_window_and_optimizer_interval tests\test_support_modules.py::test_message_bus_covers_flows -q`
  passed with `TMP`/`TEMP` redirected to the workspace.
- `go fmt ./...` passed under `services/message-gateway`.
- `go test ./...` passed under `services/message-gateway`.
- `git diff --check -- bus\shadow_gateway.py session\manager.py bootstrap\tools.py docs\sdd services\message-gateway tests\test_shadow_gateway.py`
  passed.

Decision:

- Proceed as a Phase 2.1 visibility improvement before moving any routing to Go.
