# Dashboard Agent Job Metrics Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Agent Job Metrics`
  - 校验 panel 资产包含 `No job types sampled`
  - 校验 panel 资产包含 `No recent agent-job dead letters sampled`
  - 校验 runtime overview payload 中 `agent_job_metrics` card 和关键 detail 字段仍可读
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.agent_job_metrics_table`
