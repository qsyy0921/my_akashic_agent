# Review: Phase 8.67 Outbox Metrics

Spec:
- `docs/sdd/specs/agent-gateway/006-outbox-delivery-retry.md`
- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

Implementation summary:
- Added Go `OutboxMetricsService` under `services/agent-runtime/app/service`.
- Added query and inbound port types for `OutboxMetricsView`.
- Added HTTP endpoint `GET /v1/outbox-metrics`.
- Metrics summarize sampled deliveries by status/channel kind, recent lifecycle
  throughput, current dead-letter totals, and recent dead-letter samples.
- Runtime overview dashboard now reads the Go metrics endpoint and renders it
  as an operational card.

Tests run:
- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-outbox-metrics`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py tests\test_outbox_dashboard_plugin.py -q --basetemp .tmp\pytest-outbox-metrics-dashboard-regression`
- `git diff --check`

Findings:
- This slice keeps delivery metric semantics in Go where the outbox state and
  lifecycle event stream are authoritative.
- Python dashboard code only normalizes the Go view for display and does not
  compute the primary throughput/dead-letter metric.
- The endpoint is read-only and does not lease deliveries, send platform
  messages, or acknowledge NATS messages.

Decision:
Accept as the bounded operational sample for outbox delivery metrics.

Follow-ups:
- Add persisted long-window metrics if the bounded event sample becomes too
  small for production operations.
