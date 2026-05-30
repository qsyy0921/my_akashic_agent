# Review: send ledger dashboard diagnostics

Spec:
- Continue moving stable runtime infrastructure to Go while keeping Python as the
  AI/tool worker layer.
- Make the Go-owned send ledger observable from the dashboard before expanding
  bot-to-bot loop policies.

Implementation summary:
- Added `plugins/send_ledger/dashboard.py` as a dashboard adapter over Go
  `agent-runtime` endpoints:
  - `GET /v1/send-ledger/records`
  - `GET /v1/send-ledger/recent`
- Added a `Send Ledger` dashboard panel for filtering recent send records by
  bot account, conversation, content hash, and free text.
- Added a detail-side recent-check form so an operator can verify whether a
  candidate message would hit the Go loop guard ledger window.
- Added dashboard plugin tests for runtime proxy behavior and panel asset
  exposure.

Tests run:
- `node scripts\build-plugins.mjs`
- `uv run pytest tests\test_send_ledger_dashboard_plugin.py tests\test_agent_jobs_dashboard_plugin.py -q`
- `uv run pytest tests\test_dashboard_api.py tests\test_shadow_audit_plugin.py -q`
- `C:\Users\qsyy0921\.codex\tools\go1.26.3\bin\go.exe test ./...` under
  `services/agent-runtime`
- Live runtime smoke:
  - `GET http://127.0.0.1:8780/healthz`
  - `GET http://127.0.0.1:8780/v1/send-ledger/records?limit=5`

Findings:
- This change is read-only against the Go runtime and does not alter QQ,
  Telegram, group observation, or sending behavior.
- The panel preserves the current runtime URL fallback chain:
  `AKASHIC_AGENT_RUNTIME_URL`, `AKASHIC_RUNTIME_BASE_URL`,
  `AKASHIC_GATEWAY_BASE_URL`, `AKASHIC_AGENT_GATEWAY_URL`, then
  `http://127.0.0.1:8780`.

Decision:
- Approved as an observability slice. It improves verification for Go-owned
  loop control without migrating additional behavior yet.

Follow-ups:
- Add loop-guard hit counters after Go owns more inbound filtering.
- Link send-ledger records to message/outbox events once Go owns the unified
  message event store.
