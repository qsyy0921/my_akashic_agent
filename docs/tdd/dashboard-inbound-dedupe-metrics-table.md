# Dashboard Inbound Dedupe Metrics Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Inbound Dedupe`
  - 校验 panel 资产包含 `No dedupe scopes sampled`
  - 校验 panel 资产包含 `No dedupe notes sampled`
  - 校验 panel 资产包含 `Duplicate Seen`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.inbound_dedupe_metrics_table`
