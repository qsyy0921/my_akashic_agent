# Dashboard Outbox Metrics Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Outbox Metrics`
  - 校验 panel 资产包含 `No recent dead letters sampled`
  - 校验 panel 资产包含 dead-letter 当前计数文案
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.outbox_metrics_table`
