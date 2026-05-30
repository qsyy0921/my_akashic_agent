# Review: Phase 8.70 Runtime Overview Aggregate

Spec:
- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

Implementation summary:
- Added Go query/view types for `RuntimeOverviewView`.
- Added app port `RuntimeOverviewViewer`.
- Added `RuntimeOverviewService`, which combines queue backend, delivery
  adapter, send ledger, inbox, agent job, outbox, and knowledge diagnostics.
- Added HTTP endpoint `GET /v1/runtime-overview`.
- Registered the aggregate in `cmd/agent-runtime`.
- Updated the Python runtime overview dashboard to prefer the Go aggregate and
  keep the previous multi-endpoint fallback path.

Tests run:
- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-go-aggregate`

Findings:
- This moves deterministic runtime summary/card semantics into Go, where the
  underlying lifecycle and metrics state already lives.
- Python still performs UI normalization and compatibility fallback only.
- The endpoint is read-only and does not send QQ/Telegram messages, lease
  jobs, acknowledge NATS messages, or mutate runtime state.

Decision:
Accept as the aggregate control-plane read model for runtime overview.

Follow-ups:
- After real QQ/NapCat live smoke passes, use the aggregate to confirm adapter
  enabled state before adding QQ channel aliases to outbound cutover config.
