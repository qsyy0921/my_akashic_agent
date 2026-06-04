# Dashboard Delivery Adapters Table TDD

## Tests

- 扩展 `tests/test_runtime_overview_dashboard_plugin.py`
  - 校验 panel 资产包含 `Delivery Adapters`
  - 校验 panel 资产包含 `No delivery adapters sampled`
  - 校验 panel 资产包含 `Access Token`
  - 校验 panel 资产包含 `Adapter Health`
- 扩展 `scripts/verify-go-migration-goal.ps1`
  - 新增 `dashboard_read_models.delivery_adapters_table`
