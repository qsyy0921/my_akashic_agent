# Review: outbox lease control plane

Spec:
- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`
- Continue moving stable outbound delivery infrastructure to Go while Python
  remains the compatibility platform sender.

Implementation summary:
- Added Go-owned outbox leasing through `POST /v1/outbox/lease-next`.
- Extended `OutboxDelivery` with `lease_owner` and `lease_expires_at` so future
  QQ/Telegram dispatch workers can safely claim work without double sending.
- Added leaseable lookup to both in-memory and file-backed outbox repositories.
- Updated dashboard outbox diagnostics to preserve and display lease metadata,
  and to proxy a manual `lease-next` action for smoke testing.

Tests run:
- `node scripts\build-plugins.mjs`
- `go test ./...` under `services/agent-runtime`
- `uv run pytest tests\test_outbox_dashboard_plugin.py -q` with `TMP/TEMP`
  redirected to `E:\agent\akashic\.tmp\pytest-fixed`
- `go build ./cmd/agent-runtime` under `services/agent-runtime`
- `node --check plugins\outbox\dashboard_panel.js`
- `uv run python -m compileall plugins\outbox\dashboard.py tests\test_outbox_dashboard_plugin.py`

Findings:
- This is still a control-plane migration. It does not replace Python QQ or
  Telegram SDK dispatch yet.
- Lease ownership is intentionally coarse grained by `worker_id`; adapter
  implementations should use stable worker ids such as `qq-dispatcher-1049511700`.

Decision:
- Approved to proceed as the next Go migration step because it gives future
  platform dispatchers a durable claim/retry boundary without changing current
  production send behavior.

Follow-ups:
- Add an actual Go-side dispatcher worker or Python compatibility worker that
  consumes `/v1/outbox/lease-next`.
- Add typed platform failure categories once platform adapters report structured
  send errors.
