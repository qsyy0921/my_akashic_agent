# Dashboard Inbox Metrics Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Inbox Metrics`
  - 校验 panel 资产包含 `No conversation metrics sampled`
  - 校验 panel 资产包含 `No recent inbox events sampled`
  - 校验 panel 资产包含 `Latest Seq`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.inbox_metrics_table`
