## SDD Review: Agent Job Dashboard Plugin Integration

### Scope

- Add dashboard visibility for Go-owned `AgentJob` lifecycle state.
- Expose `/api/dashboard/agent-jobs` query/lookup/retry/cancel endpoints.
- Add dashboard plugin `plugins/agent_jobs` for list + detail + batch control.
- Add tests for:
  - gateway-driven list/筛选/详情
  - retry/cancel action pass-through
  - plugin static asset exposure

### Design Notes

- `plugins/agent_jobs/dashboard.py` follows existing plugin contract:
  `register(app, plugin_dir, workspace)`.
- Gateway client behavior is standardized through wrapper methods:
  list jobs, get job, action jobs.
- Plugin UI uses lightweight in-panel filters and batch actions to avoid blocking the
  Python runtime path.

### Acceptance Check

- Dashboard list can query Go `/v1/jobs` and handle result wrappers.
- Detail and action endpoints are exposed through dashboard routes and reuse the same
  code path.
- When Go not available, API surfaces `502` with message for prompt troubleshooting.

### Status

- Completed for this slice. No architectural deviations from existing
  `dashboard_api` plugin loading.
