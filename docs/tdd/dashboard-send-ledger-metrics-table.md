# Dashboard Send Ledger Metrics Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Send Ledger Metrics`
  - 校验 panel 资产包含 `No repeated hashes sampled`
  - 校验 panel 资产包含 `No recent send ledger records sampled`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.send_ledger_metrics_table`
