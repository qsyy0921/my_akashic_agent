# Review: Phase 8.69 Send Ledger Metrics

Spec:
- `docs/sdd/specs/agent-gateway/022-send-ledger-metrics.md`
- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

Implementation summary:
- Added Go `SendLedgerService.Metrics`.
- Added query types for `SendLedgerMetricsView`.
- Added HTTP endpoint `GET /v1/send-ledger/metrics`.
- Metrics summarize sampled send records by bot account, conversation, unique
  content hash, repeated content hash, and recent records.
- Runtime overview dashboard now reads the Go metrics endpoint and renders a
  loop-guard observability card.

Tests run:
- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-send-ledger-metrics`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py tests\test_send_ledger_dashboard_plugin.py -q --basetemp .tmp\pytest-send-ledger-metrics-regression`
- `git diff --check`

Findings:
- This slice keeps recent-send metric semantics in Go where the send ledger is
  authoritative.
- Python dashboard code only normalizes the Go view for display and does not
  compute the primary bot/conversation/repeated-hash metrics.
- The endpoint is read-only and does not record sends, send platform messages,
  or mutate loop-guard state.

Decision:
Accept as a bounded operational sample for bot-to-bot loop guard observability.

Follow-ups:
- Add explicit loop-guard hit counters after Go owns more inbound filtering
  decisions.
