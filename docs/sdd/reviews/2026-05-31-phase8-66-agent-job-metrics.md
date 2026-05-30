# Review: Phase 8.66 Agent Job Metrics

Spec:
- `docs/sdd/specs/agent-gateway/008-agent-job-orchestration.md`
- `docs/sdd/specs/agent-gateway/014-runtime-dashboard-overview.md`

Implementation summary:
- Added Go `AgentJobMetricsService` under `services/agent-runtime/app/service`.
- Added query and inbound port types for `AgentJobMetricsView`.
- Added HTTP endpoint `GET /v1/job-metrics`.
- Metrics summarize sampled jobs by status/type, recent lifecycle throughput,
  current dead-letter totals, and recent dead-letter samples.
- Runtime overview dashboard now reads the Go metrics endpoint and renders it as
  an operational card.

Tests run:
- `go test ./app/service ./trigger/http`
- `go test ./...`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py -q --basetemp .tmp\pytest-runtime-overview-job-metrics`
- `uv run pytest tests\test_runtime_overview_dashboard_plugin.py tests\test_agent_jobs_dashboard_plugin.py -q --basetemp .tmp\pytest-agent-job-metrics-dashboard-regression`
- `git diff --check`

Findings:
- This slice keeps metric semantics in Go where the job state and lifecycle
  event stream are authoritative.
- Python dashboard code only normalizes the Go view for display and does not
  compute the primary throughput/dead-letter metric.
- The endpoint is read-only and does not lease jobs, execute Python workers, or
  acknowledge NATS messages.

Decision:
Accept as the first bounded operational sample for `agent_job` metrics.

Follow-ups:
- Add persisted long-window metrics if the bounded event sample becomes too
  small for production operations.
