# Dashboard Observe Capture Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Observe Capture`
  - 校验 panel 资产包含 `No observe capture targets sampled`
  - 校验 panel 资产包含 `No observe-capture notes sampled`
  - 校验 panel 资产包含 `Content Ready`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.observe_capture_table`
