# Review: Phase 8.68 Inbox Metrics

Spec:
- `docs/sdd/specs/agent-gateway/009-inbox-raw-message-store.md`
- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

Implementation summary:
- Added Go `InboxMetricsService` under `services/agent-runtime/app/service`.
- Added query and inbound port types for `InboxMetricsView`.
- Added HTTP endpoint `GET /v1/inbox-metrics`.
- Metrics summarize sampled raw inbox messages by channel kind, conversation,
  decision action, sender kind, observe-only state, attachment capture, unique
  senders, and latest `metadata.seq` cursor per conversation.
- Runtime overview dashboard now reads the Go metrics endpoint and renders it
  as an operational card.

Tests run:
- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-inbox-metrics`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py tests\test_agent_runtime_inbox_source.py tests\test_sdd_contract_fixtures.py -q --basetemp .tmp\pytest-inbox-metrics-regression`
- `git diff --check`

Findings:
- This slice keeps raw message collection metric semantics in Go where the
  inbox repository is authoritative.
- Python dashboard code only normalizes the Go view for display and does not
  compute the primary observe-only, attachment, sender, or cursor metrics.
- The endpoint is read-only and does not publish agent inbound work, send
  platform messages, or acknowledge queue messages.

Decision:
Accept as the bounded operational sample for inbox collection quality.

Follow-ups:
- Consider persisted long-window collection metrics if the bounded sample is
  too small for production QQ group monitoring.
