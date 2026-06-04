# Queue Topology Boundary Live Verifier TDD

## Targeted Checks

- `scripts/verify_queue_topology_boundary.py`
  - 覆盖 success / mismatch 结论归类
  - 覆盖 `agent_job_ack_owner` 不一致时的失败判定
- `scripts/verify-go-migration-goal.ps1`
  - 覆盖 unified verifier 纳入 `queue_topology` 证据
  - 覆盖 `dashboard_read_models.queue_topology_table`
- `plugins/runtime_overview/dashboard_panel.ts`
  - `Queue Topology` 空态文案保持稳定，可被 unified verifier 静态资产检查命中

## Commands

- `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\verify-queue-topology-boundary.ps1`
- `uv run pytest tests/test_verify_queue_topology_boundary.py -q`
- `uv run pytest tests/test_runtime_overview_dashboard_plugin.py tests/test_verify_queue_topology_boundary.py -q -W "ignore::starlette.exceptions.StarletteDeprecationWarning"`
- `uv run pytest tests/test_sdd_spec_index.py tests/test_sdd_governance_docs.py -q`
