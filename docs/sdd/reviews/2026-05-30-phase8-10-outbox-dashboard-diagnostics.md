# Review: outbox dashboard diagnostics

Spec:
- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`
- Continue moving stable outbound delivery control-plane visibility to Go while
  keeping Python as the compatibility platform sender.

Implementation summary:
- Added `plugins/outbox/dashboard.py` as a read/action proxy over Go
  `agent-runtime` endpoints:
  - `GET /v1/outbox`
  - `GET /v1/outbox/{event_id}`
  - `POST /v1/outbox/{event_id}/dispatching|succeeded|failed|retry`
- Added an `Outbox` dashboard panel for filtering deliveries by status, account,
  conversation, and free text.
- Added detail actions for dispatching, succeeded, failed, and retry so failed
  and dead-letter-adjacent records are inspectable from the browser boundary.
- Added dashboard plugin tests for list/detail/action proxy behavior and panel
  asset exposure.

Tests run:
- `node scripts\build-plugins.mjs`
- `uv run pytest tests\test_outbox_dashboard_plugin.py tests\test_send_ledger_dashboard_plugin.py tests\test_agent_jobs_dashboard_plugin.py -q`
- `go test ./...` under `services/agent-runtime`
- `node --check plugins\outbox\dashboard_panel.js`
- `uv run python -m compileall plugins\outbox\dashboard.py tests\test_outbox_dashboard_plugin.py`
- `uv run pytest tests\test_dashboard_api.py tests\test_shadow_audit_plugin.py -q`
- `go build -o ..\..\.tmp\bin\agent-runtime-outbox-dashboard-smoke.exe .\cmd\agent-runtime`
- Live dashboard proxy smoke against Go runtime on `127.0.0.1:8782`:
  - create `/v1/outbound`
  - list through `/api/dashboard/outbox`
  - mark failed through `/api/dashboard/outbox/{event_id}/failed`
  - retry through `/api/dashboard/outbox/{event_id}/retry`

Findings:
- This change is a dashboard/control-plane slice. It does not route production
  QQ or Telegram sends through Go platform adapters yet.
- Browser actions call the same Go outbox lifecycle endpoints used by tests and
  future delivery workers, so operational state stays Go-owned.

Decision:
- Approved as an observability and operations slice after Go outbox
  persistence. It makes the recoverable delivery state useful without changing
  send behavior.

Follow-ups:
- Add Go delivery worker lease/dequeue endpoints before actual platform adapter
  cutover.
- Add richer failure categorization once adapters report typed errors.
