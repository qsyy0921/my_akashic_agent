# Phase 8.147 Runtime Overview AgentJob Priority Plan

Spec: `docs/sdd/specs/agent-gateway/090-runtime-overview-agent-job-priority-plan.md`

## Summary

- Added optional `AgentJobPriorityPlan` dependency to Go
  `RuntimeOverviewService`.
- Runtime overview now exposes `agent_job_priority_*` summary fields and an
  `Agent Job Priority` card with full `agent_job_priority_plan` detail.
- Python dashboard now normalizes `agent_job_priority_plan` and provides stable
  summary defaults.

## Boundary Review

The slice remains read-only. It aggregates an existing Go control-plane plan but
does not create, lease, retry, cancel, complete, or fail AgentJobs. It does not
publish, lease, ack, nack, term, or mutate MQ work. It does not start Python
workers, change concurrency, call models, run OCR/VLM, generate images, extract
memory, or execute RAG.

Python only normalizes the dashboard response.

## Tests

- `go test ./app/service -run TestRuntimeOverviewServiceAggregatesGoOwnedDiagnostics -count=1 -v`
- `uv run pytest --basetemp .\.tmp\pytest tests/test_runtime_overview_dashboard_plugin.py -q`
- `go test ./...`
- `uv run python -m py_compile plugins/runtime_overview/dashboard.py tests/test_runtime_overview_dashboard_plugin.py`
- `git diff --check`

## Risks

- The priority plan is visible in aggregate views, but real priority scheduling
  is still intentionally absent. Any future automatic control requires operator
  acknowledgement, config audit, queue adapter support, and Python worker
  concurrency design.

## Decision

Accept as a read-only runtime overview/dashboard aggregation slice.
